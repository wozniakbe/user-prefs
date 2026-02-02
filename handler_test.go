package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockStore implements Store for testing.
type mockStore struct {
	prefs map[string]map[string]string // userID -> prefs
	err   error
}

func newMockStore() *mockStore {
	return &mockStore{prefs: make(map[string]map[string]string)}
}

func (m *mockStore) GetAll(_ context.Context, userID string) (map[string]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.prefs[userID], nil
}

func (m *mockStore) Get(_ context.Context, userID, key string) (string, bool, error) {
	if m.err != nil {
		return "", false, m.err
	}
	p := m.prefs[userID]
	if p == nil {
		return "", false, nil
	}
	v, ok := p[key]
	return v, ok, nil
}

func (m *mockStore) ReplaceAll(_ context.Context, userID string, prefs map[string]string) error {
	if m.err != nil {
		return m.err
	}
	m.prefs[userID] = prefs
	return nil
}

func (m *mockStore) Update(_ context.Context, userID string, prefs map[string]string) (map[string]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	existing := m.prefs[userID]
	if existing == nil {
		existing = make(map[string]string)
	}
	for k, v := range prefs {
		existing[k] = v
	}
	m.prefs[userID] = existing
	return existing, nil
}

func (m *mockStore) DeleteAll(_ context.Context, userID string) error {
	if m.err != nil {
		return m.err
	}
	delete(m.prefs, userID)
	return nil
}

func (m *mockStore) Delete(_ context.Context, userID, key string) error {
	if m.err != nil {
		return m.err
	}
	if p := m.prefs[userID]; p != nil {
		delete(p, key)
	}
	return nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
}

// withClaims returns a request with JWT claims set in context.
func withClaims(r *http.Request, sub string) *http.Request {
	ctx := context.WithValue(r.Context(), claimsKey, Claims{Subject: sub})
	return r.WithContext(ctx)
}

func TestGetAll_Empty(t *testing.T) {
	store := newMockStore()
	h := NewPreferencesHandler(store, testLogger())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/users/{userId}/preferences", h.GetAll)

	req := httptest.NewRequest("GET", "/api/v1/users/user1/preferences", nil)
	req = withClaims(req, "user1")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp PreferencesResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.UserID != "user1" {
		t.Fatalf("expected userId user1, got %s", resp.UserID)
	}
	if len(resp.Preferences) != 0 {
		t.Fatalf("expected empty prefs, got %v", resp.Preferences)
	}
}

func TestReplaceAllAndGetAll(t *testing.T) {
	store := newMockStore()
	h := NewPreferencesHandler(store, testLogger())

	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v1/users/{userId}/preferences", h.ReplaceAll)
	mux.HandleFunc("GET /api/v1/users/{userId}/preferences", h.GetAll)

	// PUT preferences
	body := bytes.NewBufferString(`{"theme":"dark","lang":"en"}`)
	req := httptest.NewRequest("PUT", "/api/v1/users/user1/preferences", body)
	req = withClaims(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PUT: expected 200, got %d", w.Code)
	}

	// GET preferences
	req = httptest.NewRequest("GET", "/api/v1/users/user1/preferences", nil)
	req = withClaims(req, "user1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var resp PreferencesResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Preferences["theme"] != "dark" {
		t.Fatalf("expected theme=dark, got %s", resp.Preferences["theme"])
	}
	if resp.Preferences["lang"] != "en" {
		t.Fatalf("expected lang=en, got %s", resp.Preferences["lang"])
	}
}

func TestGetOne(t *testing.T) {
	store := newMockStore()
	store.prefs["user1"] = map[string]string{"theme": "dark"}
	h := NewPreferencesHandler(store, testLogger())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/users/{userId}/preferences/{key}", h.GetOne)

	req := httptest.NewRequest("GET", "/api/v1/users/user1/preferences/theme", nil)
	req = withClaims(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp SinglePrefResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Key != "theme" || resp.Value != "dark" {
		t.Fatalf("expected theme=dark, got %s=%s", resp.Key, resp.Value)
	}
}

func TestGetOne_NotFound(t *testing.T) {
	store := newMockStore()
	h := NewPreferencesHandler(store, testLogger())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/users/{userId}/preferences/{key}", h.GetOne)

	req := httptest.NewRequest("GET", "/api/v1/users/user1/preferences/missing", nil)
	req = withClaims(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestPatchPrefs(t *testing.T) {
	store := newMockStore()
	store.prefs["user1"] = map[string]string{"theme": "dark"}
	h := NewPreferencesHandler(store, testLogger())

	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/users/{userId}/preferences", h.PatchPrefs)

	body := bytes.NewBufferString(`{"lang":"en"}`)
	req := httptest.NewRequest("PATCH", "/api/v1/users/user1/preferences", body)
	req = withClaims(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp PreferencesResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Preferences["theme"] != "dark" {
		t.Fatalf("expected theme=dark after patch, got %s", resp.Preferences["theme"])
	}
	if resp.Preferences["lang"] != "en" {
		t.Fatalf("expected lang=en after patch, got %s", resp.Preferences["lang"])
	}
}

func TestDeleteAll(t *testing.T) {
	store := newMockStore()
	store.prefs["user1"] = map[string]string{"theme": "dark"}
	h := NewPreferencesHandler(store, testLogger())

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/users/{userId}/preferences", h.DeleteAll)

	req := httptest.NewRequest("DELETE", "/api/v1/users/user1/preferences", nil)
	req = withClaims(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	if _, exists := store.prefs["user1"]; exists {
		t.Fatal("expected user1 prefs to be deleted")
	}
}

func TestDeleteOne(t *testing.T) {
	store := newMockStore()
	store.prefs["user1"] = map[string]string{"theme": "dark", "lang": "en"}
	h := NewPreferencesHandler(store, testLogger())

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/users/{userId}/preferences/{key}", h.DeleteOne)

	req := httptest.NewRequest("DELETE", "/api/v1/users/user1/preferences/theme", nil)
	req = withClaims(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	if _, exists := store.prefs["user1"]["theme"]; exists {
		t.Fatal("expected theme to be deleted")
	}
	if store.prefs["user1"]["lang"] != "en" {
		t.Fatal("expected lang to still exist")
	}
}

func TestAuthorize_Forbidden(t *testing.T) {
	store := newMockStore()
	h := NewPreferencesHandler(store, testLogger())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/users/{userId}/preferences", h.GetAll)

	req := httptest.NewRequest("GET", "/api/v1/users/user1/preferences", nil)
	req = withClaims(req, "other-user") // different user
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestStoreError(t *testing.T) {
	store := newMockStore()
	store.err = fmt.Errorf("database unavailable")
	h := NewPreferencesHandler(store, testLogger())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/users/{userId}/preferences", h.GetAll)

	req := httptest.NewRequest("GET", "/api/v1/users/user1/preferences", nil)
	req = withClaims(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestReplaceAll_InvalidJSON(t *testing.T) {
	store := newMockStore()
	h := NewPreferencesHandler(store, testLogger())

	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v1/users/{userId}/preferences", h.ReplaceAll)

	body := bytes.NewBufferString(`not json`)
	req := httptest.NewRequest("PUT", "/api/v1/users/user1/preferences", body)
	req = withClaims(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
