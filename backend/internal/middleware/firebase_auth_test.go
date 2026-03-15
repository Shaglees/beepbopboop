package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
)

func TestFirebaseAuth_DevMode_ValidHeader(t *testing.T) {
	handler := middleware.FirebaseAuth(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid := middleware.FirebaseUIDFromContext(r.Context())
		if uid != "test-user-123" {
			t.Errorf("expected uid test-user-123, got %s", uid)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/feed", nil)
	req.Header.Set("Authorization", "Bearer test-user-123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestFirebaseAuth_DevMode_MissingHeader(t *testing.T) {
	handler := middleware.FirebaseAuth(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/feed", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
