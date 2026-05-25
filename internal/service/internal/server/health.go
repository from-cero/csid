package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type healthResponse struct {
	Status string `json:"status"`
}

// handleLiveness responds to Kubernetes liveness probes.
// Returns 200 as long as the process is running.
func handleLiveness(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(healthResponse{Status: "ok"})
	if err != nil {
		slog.Warn("failed to write liveness response", "error", err)
	}
}

// handleReadiness responds to Kubernetes readiness probes.
// Returns 200 if the service is ready to receive traffic, otherwise returns 503.
func handleReadiness(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// * requirements for readiness (e.g. db ping, cache ping, etc.) should be checked here
	err := json.NewEncoder(w).Encode(healthResponse{Status: "ok"})
	if err != nil {
		slog.Warn("failed to write readiness response", "error", err)
	}
}
