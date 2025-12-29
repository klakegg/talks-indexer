package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// ReindexResponse represents the response for reindex operations
type ReindexResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// HandleReindexAll handles the full reindex endpoint
func (h *Handler) HandleReindexAll(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	slog.Info("starting full reindex")

	err := h.indexer.ReindexAll(ctx)
	if err != nil {
		slog.Error("failed to reindex all conferences", "error", err)
		h.writeErrorResponse(w, "failed to reindex all conferences", err)
		return
	}

	response := ReindexResponse{
		Status:  "success",
		Message: "successfully reindexed all conferences",
	}

	h.writeSuccessResponse(w, response)
	slog.Info("full reindex completed successfully")
}

// HandleReindexConference handles the reindex endpoint for a specific conference
func (h *Handler) HandleReindexConference(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract slug from path using Go 1.22+ path parameter feature
	slug := r.PathValue("slug")
	if slug == "" {
		h.writeErrorResponse(w, "conference slug is required", nil)
		return
	}

	slog.Info("starting conference reindex", "slug", slug)

	err := h.indexer.ReindexConference(ctx, slug)
	if err != nil {
		slog.Error("failed to reindex conference", "slug", slug, "error", err)
		h.writeErrorResponse(w, "failed to reindex conference", err)
		return
	}

	response := ReindexResponse{
		Status:  "success",
		Message: "successfully reindexed conference: " + slug,
	}

	h.writeSuccessResponse(w, response)
	slog.Info("conference reindex completed successfully", "slug", slug)
}

// HandleReindexTalk handles the reindex endpoint for a specific talk
func (h *Handler) HandleReindexTalk(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract talk ID from path using Go 1.22+ path parameter feature
	talkID := r.PathValue("talkId")
	if talkID == "" {
		h.writeErrorResponse(w, "talk ID is required", nil)
		return
	}

	slog.Info("starting talk reindex", "talkID", talkID)

	err := h.indexer.ReindexTalk(ctx, talkID)
	if err != nil {
		slog.Error("failed to reindex talk", "talkID", talkID, "error", err)
		h.writeErrorResponse(w, "failed to reindex talk", err)
		return
	}

	response := ReindexResponse{
		Status:  "success",
		Message: "successfully reindexed talk: " + talkID,
	}

	h.writeSuccessResponse(w, response)
	slog.Info("talk reindex completed successfully", "talkID", talkID)
}

// writeSuccessResponse writes a successful JSON response
func (h *Handler) writeSuccessResponse(w http.ResponseWriter, response ReindexResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("failed to encode success response", "error", err)
	}
}

// writeErrorResponse writes an error JSON response
func (h *Handler) writeErrorResponse(w http.ResponseWriter, message string, err error) {
	response := ReindexResponse{
		Status:  "error",
		Message: message,
	}

	if err != nil {
		response.Message = message + ": " + err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("failed to encode error response", "error", err)
	}
}
