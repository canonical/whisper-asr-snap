package server

import (
	"encoding/json"
	"net/http"
)

// modelOwner identifies this adapter as the "owner" of the models it
// advertises, matching the "owned_by" field OpenAI clients expect.
const modelOwner = "myna-adapter"

// Model describes a single entry in the OpenAI-compatible /v1/models
// response.
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ModelList is the OpenAI-compatible response body for /v1/models.
type ModelList struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

// handleModels serves an OpenAI-compatible list of the models allowed by
// the --allowed-models CLI flag.
func (s *WebSocketServer) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	created := s.startTime.Unix()
	data := make([]Model, 0, len(s.allowedModels))
	for _, id := range s.allowedModels {
		data = append(data, Model{
			ID:      id,
			Object:  "model",
			Created: created,
			OwnedBy: modelOwner,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ModelList{Object: "list", Data: data})
}
