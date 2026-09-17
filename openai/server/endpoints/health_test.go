package endpoints

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"
)

// TestHealth_NoChecker verifies that /health reports "ok" with no checks
// when no BackendChecker is configured.
func TestHealth_NoChecker(t *testing.T) {
	startTime := time.Now().Add(-5 * time.Second)
	handler := Health(startTime, nil)

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var got HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshalling response: %v", err)
	}

	if got.Status != "ok" {
		t.Fatalf("expected status %q, got %q", "ok", got.Status)
	}
	if got.Checks != nil {
		t.Fatalf("expected no checks, got %+v", got.Checks)
	}
	if got.UptimeSeconds < 5 {
		t.Fatalf("expected uptime_seconds >= 5, got %f", got.UptimeSeconds)
	}
	if got.Timestamp == "" {
		t.Fatal("expected a non-empty timestamp")
	}
}

// TestHealth_BackendReachable verifies that a successful backend check is
// reported under checks.backend with status "ok" and status code 200.
func TestHealth_BackendReachable(t *testing.T) {
	handler := Health(time.Now(), func(ctx context.Context) error { return nil })

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var got HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshalling response: %v", err)
	}

	if got.Status != "ok" {
		t.Fatalf("expected status %q, got %q", "ok", got.Status)
	}
	backend, ok := got.Checks["backend"]
	if !ok {
		t.Fatal("expected a backend check")
	}
	if backend.Status != "ok" {
		t.Fatalf("expected backend status %q, got %q", "ok", backend.Status)
	}
	if backend.LatencyMs == nil {
		t.Fatal("expected a non-nil latency_ms for a successful check")
	}
	if backend.Error != "" {
		t.Fatalf("expected no error, got %q", backend.Error)
	}
}

// TestHealth_BackendUnreachable verifies that a failing backend check is
// reported as "degraded" with a 503 status and the error message included.
func TestHealth_BackendUnreachable(t *testing.T) {
	wantErr := errors.New("connection refused")
	handler := Health(time.Now(), func(ctx context.Context) error { return wantErr })

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != 503 {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}

	var got HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshalling response: %v", err)
	}

	if got.Status != "degraded" {
		t.Fatalf("expected status %q, got %q", "degraded", got.Status)
	}
	backend, ok := got.Checks["backend"]
	if !ok {
		t.Fatal("expected a backend check")
	}
	if backend.Status != "error" {
		t.Fatalf("expected backend status %q, got %q", "error", backend.Status)
	}
	if backend.Error != wantErr.Error() {
		t.Fatalf("expected error %q, got %q", wantErr.Error(), backend.Error)
	}
}
