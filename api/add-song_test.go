package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAddSongHandler_MissingAPIKey(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/add-song", nil)
	recorder := httptest.NewRecorder()

	AddSongHandler(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}
