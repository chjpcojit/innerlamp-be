package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"chjpcoj.com.innerlamp/pkg/utils"
)

// corsHeaders is the exact set CORS hardcodes on every response (AC3).
var corsHeaders = map[string]string{
	"Access-Control-Allow-Origin":  "*",
	"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
	"Access-Control-Allow-Headers": "Content-Type, Authorization",
}

func assertCORSHeaders(t *testing.T, h http.Header) {
	t.Helper()
	for name, want := range corsHeaders {
		if got := h.Get(name); got != want {
			t.Errorf("header %s = %q, want %q", name, got, want)
		}
	}
}

func TestCORS_PreflightIsEnvelope(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	})

	rr := httptest.NewRecorder()
	CORS(next).ServeHTTP(rr, httptest.NewRequest(http.MethodOptions, "/v1/auth/login", nil))

	if nextCalled {
		t.Fatal("preflight must not reach the next handler")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	assertCORSHeaders(t, rr.Header())

	var resp utils.ApiResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("body is not the ApiResponse envelope: %v (%q)", err, rr.Body.String())
	}
	if !resp.Success || resp.Data != nil || resp.Error != nil {
		t.Fatalf("unexpected preflight envelope: %q", rr.Body.String())
	}
}

func TestCORS_NonOptionsPassesThrough(t *testing.T) {
	const body = `{"ok":true}`
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(body))
	})

	rr := httptest.NewRecorder()
	CORS(next).ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/auth/login", nil))

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (unchanged)", rr.Code)
	}
	if got := rr.Body.String(); got != body {
		t.Fatalf("body = %q, want %q (unchanged)", got, body)
	}
}

func TestCORS_NonOptionsSetsHeaders(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	rr := httptest.NewRecorder()
	CORS(next).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/auth/login", nil))

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rr.Code)
	}
	assertCORSHeaders(t, rr.Header())
}

func TestRecovery_PanicUsesEnvelope(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
	var resp utils.ApiResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("body is not the ApiResponse envelope: %v (%q)", err, rr.Body.String())
	}
	if resp.Success || resp.Error == nil || resp.Error.Code != "INTERNAL_ERROR" || resp.Error.Type != utils.ErrorTypeSystem {
		t.Fatalf("unexpected envelope: %q", rr.Body.String())
	}
}
