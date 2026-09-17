package endpoints

import "net/http"

// Health returns the handler for the catch-all "/" liveness check.
func Health() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{\"status\":\"ok\"}\n"))
	}
}
