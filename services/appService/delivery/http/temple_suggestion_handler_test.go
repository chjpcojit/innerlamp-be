package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

type fakeTempleSuggestionUsecase struct {
	got   *domain.TempleSuggestionInput
	out   *domain.TempleSuggestion
	err   error
	calls int
}

func (f *fakeTempleSuggestionUsecase) Create(_ context.Context, in domain.TempleSuggestionInput) (*domain.TempleSuggestion, error) {
	f.calls++
	f.got = &in
	return f.out, f.err
}

type suggestionEnvelope struct {
	Success bool `json:"success"`
	Data    *struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
	} `json:"data"`
	Error *struct {
		Code string `json:"code"`
	} `json:"error"`
	Details map[string]any `json:"details"`
}

func postSuggestion(t *testing.T, uc templeSuggestionUsecase, body string) (*httptest.ResponseRecorder, suggestionEnvelope) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/temple-suggestions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	NewTempleSuggestionHandler(uc).Create(w, req)
	var env suggestionEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body: %s", w.Body.String())
	return w, env
}

func TestTempleSuggestionHandler_Create_Success(t *testing.T) {
	uc := &fakeTempleSuggestionUsecase{out: &domain.TempleSuggestion{ID: 12, Status: domain.TempleSuggestionPending}}

	w, env := postSuggestion(t, uc, `{"name":"Chùa A","province":"Huế","address":"Đường 1"}`)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.True(t, env.Success)
	require.NotNil(t, env.Data)
	assert.Equal(t, int64(12), env.Data.ID)
	assert.Equal(t, "pending", env.Data.Status)
	assert.Equal(t, domain.TempleSuggestionInput{Name: "Chùa A", Province: "Huế", Address: "Đường 1"}, *uc.got)
}

func TestTempleSuggestionHandler_Create_ResponseHasOnlyIDAndStatus(t *testing.T) {
	uc := &fakeTempleSuggestionUsecase{out: &domain.TempleSuggestion{
		ID: 1, Name: "secret-name", Province: "p", Address: "a", Status: domain.TempleSuggestionPending,
	}}

	w, _ := postSuggestion(t, uc, `{"name":"x","province":"y","address":"z"}`)

	var raw struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &raw))
	assert.Len(t, raw.Data, 2)
	assert.Contains(t, raw.Data, "id")
	assert.Contains(t, raw.Data, "status")
}

func TestTempleSuggestionHandler_Create_ValidationErrorNamesField(t *testing.T) {
	uc := &fakeTempleSuggestionUsecase{err: &domain.TempleSuggestionValidationError{Field: "address", Message: "address is required"}}

	w, env := postSuggestion(t, uc, `{"name":"x","province":"y","address":""}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	require.NotNil(t, env.Error)
	assert.Equal(t, "VALIDATION_ERROR", env.Error.Code)
	assert.Equal(t, "address", env.Details["field"])
}

func TestTempleSuggestionHandler_Create_BadBodyNeverReachesUsecase(t *testing.T) {
	for name, body := range map[string]string{
		"not json":      `nope`,
		"unknown field": `{"name":"x","province":"y","address":"z","status":"created"}`,
		"trailing data": `{"name":"x","province":"y","address":"z"} {}`,
		"empty body":    ``,
	} {
		t.Run(name, func(t *testing.T) {
			uc := &fakeTempleSuggestionUsecase{}

			w, env := postSuggestion(t, uc, body)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.False(t, env.Success)
			assert.Zero(t, uc.calls)
		})
	}
}

func TestTempleSuggestionHandler_Create_InfrastructureErrorDoesNotLeak(t *testing.T) {
	uc := &fakeTempleSuggestionUsecase{err: errors.New("pq: connection refused to 10.1.2.3")}

	w, env := postSuggestion(t, uc, `{"name":"x","province":"y","address":"z"}`)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	require.NotNil(t, env.Error)
	assert.Equal(t, "INTERNAL_ERROR", env.Error.Code)
	assert.NotContains(t, w.Body.String(), "10.1.2.3")
}
