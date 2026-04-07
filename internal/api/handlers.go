package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/foonly/bookmarks-api/internal/models"
	"github.com/foonly/bookmarks-api/internal/store"
	"github.com/go-chi/chi/v5"
)

const (
	// MaxPayloadSize defines the maximum allowed size for the encrypted blob (1MB)
	MaxPayloadSize = 1024 * 1024
)

type Handler struct {
	store store.Store
}

func NewHandler(s store.Store) *Handler {
	return &Handler{store: s}
}

// GetLatest handles GET /api/v1/sync/:id
func (h *Handler) GetLatest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing sync id", http.StatusBadRequest)
		return
	}

	blob, err := h.store.GetLatestBlob(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "sync id not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.respondWithJSON(w, http.StatusOK, blob)
}

// Upload handles POST /api/v1/sync/:id
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing sync id", http.StatusBadRequest)
		return
	}

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, MaxPayloadSize)

	var req models.SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body or payload too large", http.StatusBadRequest)
		return
	}

	if req.Data == "" {
		http.Error(w, "data blob is required", http.StatusBadRequest)
		return
	}

	err := h.store.SaveBlob(r.Context(), id, req.Data)
	if err != nil {
		http.Error(w, "failed to save sync data", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// GetHistory handles GET /api/v1/sync/:id/history
func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing sync id", http.StatusBadRequest)
		return
	}

	history, err := h.store.GetHistory(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "no history found for this id", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.respondWithJSON(w, http.StatusOK, history)
}

// GetVersion handles GET /api/v1/sync/:id/:timestamp
func (h *Handler) GetVersion(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tsStr := chi.URLParam(r, "timestamp")

	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid timestamp format", http.StatusBadRequest)
		return
	}

	blob, err := h.store.GetBlobAtTimestamp(r.Context(), id, ts)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "version not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.respondWithJSON(w, http.StatusOK, blob)
}

func (h *Handler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
