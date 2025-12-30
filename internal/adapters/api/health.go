package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status string `json:"status"`
}

// HandleHealth handles the health check endpoint
func (a *Adapter) HandleHealth(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status: "ok",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("failed to encode health response", "error", err)
	}
}
