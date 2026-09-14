package endpoints

import (
	"encoding/json"
	"net/http"
	"time"
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

// Models returns the /v1/models handler, listing allowedModels in an
// OpenAI-compatible response. created is used as each model's "created"
// timestamp.
func Models(allowedModels []string, created time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		createdUnix := created.Unix()
		data := make([]Model, 0, len(allowedModels))
		for _, id := range allowedModels {
			data = append(data, Model{
				ID:      id,
				Object:  "model",
				Created: createdUnix,
				OwnedBy: modelOwner,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ModelList{Object: "list", Data: data})
	}
}
