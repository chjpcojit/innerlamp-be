package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"chjpcoj.com.innerlamp/pkg/utils"
)

// testMux builds a ServeMux with a route that replies a fixed status/body and
// a route that panics, so newHandler can be exercised without a database or
// config (this package has no other tests, so the global log redirection in
// the log tests cannot race another test's logger).
func testMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/ok", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte(`{"success":true,"data":{"hello":"world"}}`))
	})
	mux.HandleFunc("GET /v1/boom", func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})
	return mux
}

func serve(mux *http.ServeMux, method, path string) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	newHandler(mux).ServeHTTP(rr, httptest.NewRequest(method, path, nil))
	return rr
}

func assertCORSHeaders(t *testing.T, h http.Header) {
	t.Helper()
	for name, want := range map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, POST, PUT, PATCH, DELETE, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization",
	} {
		if got := h.Get(name); got != want {
			t.Errorf("header %s = %q, want %q", name, got, want)
		}
	}
}

func decodeEnvelope(t *testing.T, body string) utils.ApiResponse {
	t.Helper()
	var resp utils.ApiResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("body is not the ApiResponse envelope: %v (%q)", err, body)
	}
	return resp
}

// AC3 + AC6: a matched route keeps its own status and body and gains the CORS
// headers.
func TestNewHandler_MatchedRouteUnchangedPlusCORS(t *testing.T) {
	rr := serve(testMux(), http.MethodGet, "/v1/ok")

	if rr.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want 418 (unchanged)", rr.Code)
	}
	if got, want := rr.Body.String(), `{"success":true,"data":{"hello":"world"}}`; got != want {
		t.Fatalf("body = %q, want %q (unchanged)", got, want)
	}
	assertCORSHeaders(t, rr.Header())
}

// AC7: an unknown path still gets the 404 NOT_FOUND envelope, through the
// fallback inside the chain.
func TestNewHandler_UnknownPathIsNotFoundEnvelope(t *testing.T) {
	rr := serve(testMux(), http.MethodGet, "/v1/nope")

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
	resp := decodeEnvelope(t, rr.Body.String())
	if resp.Success || resp.Error == nil || resp.Error.Code != "NOT_FOUND" {
		t.Fatalf("unexpected envelope: %q", rr.Body.String())
	}
	assertCORSHeaders(t, rr.Header())
}

// AC7: a known path with the wrong method still gets 405 METHOD_NOT_ALLOWED.
func TestNewHandler_WrongMethodIsMethodNotAllowedEnvelope(t *testing.T) {
	rr := serve(testMux(), http.MethodDelete, "/v1/ok")

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rr.Code)
	}
	resp := decodeEnvelope(t, rr.Body.String())
	if resp.Success || resp.Error == nil || resp.Error.Code != "METHOD_NOT_ALLOWED" {
		t.Fatalf("unexpected envelope: %q", rr.Body.String())
	}
	assertCORSHeaders(t, rr.Header())
}

// AC8: Recovery is outermost, so a panic raised inside the chain is answered
// 500 INTERNAL_ERROR and still carries the CORS headers (CORS already ran).
func TestNewHandler_PanicIsInternalErrorEnvelope(t *testing.T) {
	rr := serve(testMux(), http.MethodGet, "/v1/boom")

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rr.Code)
	}
	resp := decodeEnvelope(t, rr.Body.String())
	if resp.Success || resp.Error == nil || resp.Error.Code != "INTERNAL_ERROR" || resp.Error.Type != utils.ErrorTypeSystem {
		t.Fatalf("unexpected envelope: %q", rr.Body.String())
	}
	assertCORSHeaders(t, rr.Header())
}

// AC4 + AC10: CORS answers preflight itself with the standard envelope and
// never reaches the mux (the panic route stays silent, and the matched route
// would be 418 if it ran).
func TestNewHandler_PreflightStopsAtCORS(t *testing.T) {
	rr := serve(testMux(), http.MethodOptions, "/v1/ok")

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	resp := decodeEnvelope(t, rr.Body.String())
	if !resp.Success || resp.Data != nil || resp.Error != nil {
		t.Fatalf("unexpected preflight envelope: %q", rr.Body.String())
	}
	assertCORSHeaders(t, rr.Header())
}

// AC2 + AC9: a normal request logs a start line (method + path) and a
// completion line (duration); preflight logs nothing because CORS
// short-circuits before Logger.
func TestNewHandler_LogsNonPreflightOnly(t *testing.T) {
	var buf bytes.Buffer
	orig := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(orig) })

	serve(testMux(), http.MethodGet, "/v1/nope")
	logged := buf.String()
	if len(strings.Split(strings.TrimSpace(logged), "\n")) != 2 {
		t.Fatalf("expected 2 log lines for a normal request, got %q", logged)
	}
	if !strings.Contains(logged, `Started "GET" "/v1/nope"`) {
		t.Fatalf("start line missing method/path: %q", logged)
	}
	if !strings.Contains(logged, "Completed in ") {
		t.Fatalf("completion line missing duration: %q", logged)
	}

	buf.Reset()
	serve(testMux(), http.MethodOptions, "/v1/nope")
	if got := buf.String(); got != "" {
		t.Fatalf("preflight must not be logged, got %q", got)
	}
}
