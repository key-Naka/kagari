package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatusHandler(t *testing.T) {
	t.Run("returns all metric availability states", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/status", nil)
		response := httptest.NewRecorder()

		statusHandler(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
		}
		if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
			t.Fatalf("Content-Type = %q, want %q", contentType, "application/json")
		}
		var body map[string]availability
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(body) != 5 {
			t.Fatalf("metric count = %d, want 5: %#v", len(body), body)
		}
		for _, metric := range []string{"cpu", "memory", "disk", "network", "uptime"} {
			state, ok := body[metric]
			if !ok {
				t.Errorf("missing %q metric", metric)
				continue
			}
			if state != availabilityOperational && state != availabilityDegraded && state != availabilityUnavailable {
				t.Errorf("%s = %q, want a valid availability", metric, state)
			}
		}
	})

	t.Run("rejects unsupported methods", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/status", nil)
		response := httptest.NewRecorder()

		statusHandler(response, request)

		if response.Code != http.StatusMethodNotAllowed {
			t.Errorf("status code = %d, want %d", response.Code, http.StatusMethodNotAllowed)
		}
	})
}
