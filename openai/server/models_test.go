package server

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// TestHandleModels_ListsAllowedModels verifies that /v1/models returns an
// OpenAI-compatible response listing exactly the configured allowed models.
func TestHandleModels_ListsAllowedModels(t *testing.T) {
	s := NewWebSocketServer("127.0.0.1", 0, "")
	s.SetAllowedModels([]string{"small", "tiny"})

	req := httptest.NewRequest("GET", "/v1/models", nil)
	rec := httptest.NewRecorder()

	s.handleModels(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var got ModelList
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshalling response: %v", err)
	}

	if got.Object != "list" {
		t.Fatalf("expected object %q, got %q", "list", got.Object)
	}

	if len(got.Data) != 2 {
		t.Fatalf("expected 2 models, got %d: %+v", len(got.Data), got.Data)
	}

	ids := []string{got.Data[0].ID, got.Data[1].ID}
	if ids[0] != "small" || ids[1] != "tiny" {
		t.Fatalf("expected models [small tiny] in order, got %v", ids)
	}

	for _, m := range got.Data {
		if m.Object != "model" {
			t.Errorf("expected object %q for model %q, got %q", "model", m.ID, m.Object)
		}
		if m.OwnedBy == "" {
			t.Errorf("expected non-empty owned_by for model %q", m.ID)
		}
	}
}

// TestHandleModels_MethodNotAllowed verifies non-GET requests are rejected.
func TestHandleModels_MethodNotAllowed(t *testing.T) {
	s := NewWebSocketServer("127.0.0.1", 0, "")
	s.SetAllowedModels([]string{"small"})

	req := httptest.NewRequest("POST", "/v1/models", nil)
	rec := httptest.NewRecorder()

	s.handleModels(rec, req)

	if rec.Code != 405 {
		t.Fatalf("expected status 405, got %d", rec.Code)
	}
}
