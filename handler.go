package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// PreferencesHandler holds dependencies for preference CRUD handlers.
type PreferencesHandler struct {
	store  Store
	logger *slog.Logger
}

// NewPreferencesHandler creates a new handler with the given store and logger.
func NewPreferencesHandler(store Store, logger *slog.Logger) *PreferencesHandler {
	return &PreferencesHandler{store: store, logger: logger}
}

// authorize checks that the JWT subject matches the requested userId.
func (h *PreferencesHandler) authorize(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := r.PathValue("userId")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "missing userId")
		return "", false
	}

	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing claims")
		return "", false
	}

	if claims.Subject != userID {
		writeError(w, http.StatusForbidden, "access denied")
		return "", false
	}

	return userID, true
}

// GetAll returns all preferences for a user.
func (h *PreferencesHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authorize(w, r)
	if !ok {
		return
	}

	prefs, err := h.store.GetAll(r.Context(), userID)
	if err != nil {
		h.logger.Error("store.GetAll failed", "error", err, "userId", userID)
		writeError(w, http.StatusInternalServerError, "failed to retrieve preferences")
		return
	}

	if prefs == nil {
		prefs = make(map[string]string)
	}

	writeJSON(w, http.StatusOK, PreferencesResponse{
		UserID:      userID,
		Preferences: prefs,
	})
}

// GetOne returns a single preference by key.
func (h *PreferencesHandler) GetOne(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authorize(w, r)
	if !ok {
		return
	}

	key := r.PathValue("key")
	if key == "" {
		writeError(w, http.StatusBadRequest, "missing key")
		return
	}

	value, found, err := h.store.Get(r.Context(), userID, key)
	if err != nil {
		h.logger.Error("store.Get failed", "error", err, "userId", userID, "key", key)
		writeError(w, http.StatusInternalServerError, "failed to retrieve preference")
		return
	}

	if !found {
		writeError(w, http.StatusNotFound, "preference not found")
		return
	}

	writeJSON(w, http.StatusOK, SinglePrefResponse{Key: key, Value: value})
}

// ReplaceAll replaces all preferences for a user (PUT and POST).
func (h *PreferencesHandler) ReplaceAll(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authorize(w, r)
	if !ok {
		return
	}

	var prefs map[string]string
	if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := h.store.ReplaceAll(r.Context(), userID, prefs); err != nil {
		h.logger.Error("store.ReplaceAll failed", "error", err, "userId", userID)
		writeError(w, http.StatusInternalServerError, "failed to save preferences")
		return
	}

	writeJSON(w, http.StatusOK, PreferencesResponse{
		UserID:      userID,
		Preferences: prefs,
	})
}

// PatchPrefs partially updates preferences (merge).
func (h *PreferencesHandler) PatchPrefs(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authorize(w, r)
	if !ok {
		return
	}

	var prefs map[string]string
	if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if len(prefs) == 0 {
		writeError(w, http.StatusBadRequest, "empty preferences")
		return
	}

	merged, err := h.store.Update(r.Context(), userID, prefs)
	if err != nil {
		h.logger.Error("store.Update failed", "error", err, "userId", userID)
		writeError(w, http.StatusInternalServerError, "failed to update preferences")
		return
	}

	writeJSON(w, http.StatusOK, PreferencesResponse{
		UserID:      userID,
		Preferences: merged,
	})
}

// DeleteAll removes all preferences for a user.
func (h *PreferencesHandler) DeleteAll(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authorize(w, r)
	if !ok {
		return
	}

	if err := h.store.DeleteAll(r.Context(), userID); err != nil {
		h.logger.Error("store.DeleteAll failed", "error", err, "userId", userID)
		writeError(w, http.StatusInternalServerError, "failed to delete preferences")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteOne removes a single preference by key.
func (h *PreferencesHandler) DeleteOne(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authorize(w, r)
	if !ok {
		return
	}

	key := r.PathValue("key")
	if key == "" {
		writeError(w, http.StatusBadRequest, "missing key")
		return
	}

	if err := h.store.Delete(r.Context(), userID, key); err != nil {
		h.logger.Error("store.Delete failed", "error", err, "userId", userID, "key", key)
		writeError(w, http.StatusInternalServerError, "failed to delete preference")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
