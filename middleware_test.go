package main

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-key"

func makeToken(sub string, secret string, method jwt.SigningMethod) string {
	claims := jwt.MapClaims{"sub": sub}
	token := jwt.NewWithClaims(method, claims)
	s, _ := token.SignedString([]byte(secret))
	return s
}

func makeTokenWithExp(sub string, secret string, exp time.Time) string {
	claims := jwt.MapClaims{"sub": sub, "exp": jwt.NewNumericDate(exp)}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := token.SignedString([]byte(secret))
	return s
}

// jwtTestMux creates a mux with a single route so that PathValue is populated.
func jwtTestMux(auth func(http.HandlerFunc) http.HandlerFunc, inner http.HandlerFunc) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/users/{userId}/preferences", auth(inner))
	return mux
}

func TestJWTAuth_ValidToken(t *testing.T) {
	token := makeToken("user1", testSecret, jwt.SigningMethodHS256)
	auth := JWTAuth(testSecret, "", false)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			t.Fatal("expected claims in context")
		}
		if claims.Subject != "user1" {
			t.Fatalf("expected sub=user1, got %s", claims.Subject)
		}
		w.WriteHeader(http.StatusOK)
	})

	mux := jwtTestMux(auth, inner)
	req := httptest.NewRequest("GET", "/api/v1/users/user1/preferences", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	auth := JWTAuth(testSecret, "", false)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	mux := jwtTestMux(auth, inner)
	req := httptest.NewRequest("GET", "/api/v1/users/user1/preferences", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	auth := JWTAuth(testSecret, "", false)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	mux := jwtTestMux(auth, inner)
	req := httptest.NewRequest("GET", "/api/v1/users/user1/preferences", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestJWTAuth_WrongSecret(t *testing.T) {
	token := makeToken("user1", "wrong-secret", jwt.SigningMethodHS256)
	auth := JWTAuth(testSecret, "", false)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	mux := jwtTestMux(auth, inner)
	req := httptest.NewRequest("GET", "/api/v1/users/user1/preferences", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	token := makeTokenWithExp("user1", testSecret, time.Now().Add(-1*time.Hour))
	auth := JWTAuth(testSecret, "", false)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	mux := jwtTestMux(auth, inner)
	req := httptest.NewRequest("GET", "/api/v1/users/user1/preferences", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestJWTAuth_BadFormat(t *testing.T) {
	auth := JWTAuth(testSecret, "", false)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	mux := jwtTestMux(auth, inner)
	req := httptest.NewRequest("GET", "/api/v1/users/user1/preferences", nil)
	req.Header.Set("Authorization", "NotBearer token")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestJWTAuth_IssuerValidation(t *testing.T) {
	// Token without issuer, but middleware expects one
	claims := jwt.MapClaims{"sub": "user1"}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(testSecret))

	auth := JWTAuth(testSecret, "expected-issuer", false)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	mux := jwtTestMux(auth, inner)
	req := httptest.NewRequest("GET", "/api/v1/users/user1/preferences", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: token without matching issuer should be rejected", w.Code)
	}
}

func TestJWTAuth_DevBypass(t *testing.T) {
	auth := JWTAuth(testSecret, "", true)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			t.Fatal("expected claims in context")
		}
		if claims.Subject != "user1" {
			t.Fatalf("expected sub=user1, got %s", claims.Subject)
		}
		w.WriteHeader(http.StatusOK)
	})

	mux := jwtTestMux(auth, inner)
	req := httptest.NewRequest("GET", "/api/v1/users/user1/preferences", nil)
	// No Authorization header — bypass should skip validation
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCORS(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := CORS("https://example.com")(inner)

	// Normal request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Fatalf("expected CORS origin header, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}

	// Preflight request
	req = httptest.NewRequest("OPTIONS", "/test", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS, got %d", w.Code)
	}
}

func TestRecovery(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	handler := Recovery(logger)(inner)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestRequestLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RequestLogging(logger)(inner)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	logOutput := buf.String()
	if logOutput == "" {
		t.Fatal("expected log output, got nothing")
	}
	if !contains(logOutput, "GET") || !contains(logOutput, "/test") {
		t.Fatalf("expected log to contain method and path, got: %s", logOutput)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
