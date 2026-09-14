package endpoints

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// backendCheckTimeout bounds how long the backend reachability probe may
// take before the health endpoint gives up and reports it as unreachable.
const backendCheckTimeout = 2 * time.Second

// BackendChecker reports whether the transcription backend the adapter
// depends on is currently reachable. Implementations should respect ctx's
// deadline and return promptly without any lasting side effects (e.g. a
// plain TCP dial), since it runs on every /health request.
type BackendChecker func(ctx context.Context) error

// checkStatus is the status of a single dependency check.
type checkStatus struct {
	// Status is "ok" or "error".
	Status string `json:"status"`
	// Error is populated only when Status is "error".
	Error string `json:"error,omitempty"`
	// LatencyMs is populated only when Status is "ok". It is a pointer so a
	// genuine 0ms latency is still serialized instead of being omitted.
	LatencyMs *int64 `json:"latency_ms,omitempty"`
}

// HealthResponse is the JSON body returned by the /health endpoint.
type HealthResponse struct {
	// Status is "ok" when the process and all of its checked dependencies
	// are healthy, or "degraded" when at least one dependency check failed.
	// The process itself responding at all already implies it is alive, so
	// there is no separate "down" status.
	Status string `json:"status"`
	// Timestamp is when this check was performed, in RFC 3339 format.
	Timestamp string `json:"timestamp"`
	// UptimeSeconds is how long the server has been running.
	UptimeSeconds float64 `json:"uptime_seconds"`
	// Checks reports the status of each dependency the server relies on,
	// keyed by dependency name (currently only "backend").
	Checks map[string]checkStatus `json:"checks,omitempty"`
}

// Health returns the /health handler. startTime is used to report process
// uptime. checkBackend probes whether the transcription backend is
// reachable and is reported under the "backend" check; pass nil to omit the
// backend check entirely (e.g. in tests).
func Health(startTime time.Time, checkBackend BackendChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		resp := HealthResponse{
			Status:        "ok",
			Timestamp:     now.UTC().Format(time.RFC3339),
			UptimeSeconds: now.Sub(startTime).Seconds(),
		}

		if checkBackend != nil {
			ctx, cancel := context.WithTimeout(r.Context(), backendCheckTimeout)
			defer cancel()

			checkStart := time.Now()
			resp.Checks = map[string]checkStatus{}
			if err := checkBackend(ctx); err != nil {
				resp.Status = "degraded"
				resp.Checks["backend"] = checkStatus{Status: "error", Error: err.Error()}
			} else {
				latencyMs := time.Since(checkStart).Milliseconds()
				resp.Checks["backend"] = checkStatus{
					Status:    "ok",
					LatencyMs: &latencyMs,
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if resp.Status != "ok" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}
