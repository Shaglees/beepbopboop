package handler_test

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func setupPhotoTest(t *testing.T) (*handler.PhotoHandler, *repository.UserRepo, *repository.AgentRepo) {
	t.Helper()
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	photoRepo := repository.NewUserPhotoRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	h := handler.NewPhotoHandler(userRepo, photoRepo, agentRepo)
	return h, userRepo, agentRepo
}

func buildMultipartRequest(t *testing.T, method, url string, fieldName string, data []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile(fieldName, "photo.jpg")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fw.Write(data); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	mw.Close()

	req := httptest.NewRequest(method, url, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestPhotoHandler_UploadHeadshot(t *testing.T) {
	h, _, _ := setupPhotoTest(t)

	fakeJPEG := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10} // JPEG header bytes
	req := buildMultipartRequest(t, "PUT", "/user/photos/headshot", "photo", fakeJPEG)
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-photo-upload"))

	w := httptest.NewRecorder()
	h.UploadHeadshot(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("upload headshot: status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}

func TestPhotoHandler_GetHeadshot(t *testing.T) {
	h, _, _ := setupPhotoTest(t)
	firebaseUID := "firebase-photo-get"

	// Upload first
	fakeJPEG := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}
	uploadReq := buildMultipartRequest(t, "PUT", "/user/photos/headshot", "photo", fakeJPEG)
	uploadReq = uploadReq.WithContext(middleware.WithFirebaseUID(uploadReq.Context(), firebaseUID))
	uploadW := httptest.NewRecorder()
	h.UploadHeadshot(uploadW, uploadReq)
	if uploadW.Code != http.StatusOK {
		t.Fatalf("upload failed: %d %s", uploadW.Code, uploadW.Body.String())
	}

	// Get
	getReq := httptest.NewRequest("GET", "/user/photos/headshot", nil)
	getReq = getReq.WithContext(middleware.WithFirebaseUID(getReq.Context(), firebaseUID))
	getW := httptest.NewRecorder()
	h.GetHeadshot(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("get headshot: status = %d, want 200; body = %s", getW.Code, getW.Body.String())
	}
	ct := getW.Header().Get("Content-Type")
	if ct != "image/jpeg" {
		t.Errorf("content-type = %q, want image/jpeg", ct)
	}
	body, _ := io.ReadAll(getW.Body)
	if !bytes.Equal(body, fakeJPEG) {
		t.Errorf("response body = %v, want %v", body, fakeJPEG)
	}
}

func TestPhotoHandler_GetEmpty(t *testing.T) {
	h, _, _ := setupPhotoTest(t)

	req := httptest.NewRequest("GET", "/user/photos/headshot", nil)
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-photo-empty"))
	w := httptest.NewRecorder()
	h.GetHeadshot(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("get empty headshot: status = %d, want 404; body = %s", w.Code, w.Body.String())
	}
}

func TestPhotoHandler_Delete(t *testing.T) {
	h, _, _ := setupPhotoTest(t)
	firebaseUID := "firebase-photo-delete"

	// Upload
	fakeJPEG := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	uploadReq := buildMultipartRequest(t, "PUT", "/user/photos/headshot", "photo", fakeJPEG)
	uploadReq = uploadReq.WithContext(middleware.WithFirebaseUID(uploadReq.Context(), firebaseUID))
	uploadW := httptest.NewRecorder()
	h.UploadHeadshot(uploadW, uploadReq)
	if uploadW.Code != http.StatusOK {
		t.Fatalf("upload failed: %d %s", uploadW.Code, uploadW.Body.String())
	}

	// Delete
	deleteReq := httptest.NewRequest("DELETE", "/user/photos/headshot", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("type", "headshot")
	deleteReq = deleteReq.WithContext(
		context.WithValue(
			middleware.WithFirebaseUID(deleteReq.Context(), firebaseUID),
			chi.RouteCtxKey, rctx,
		),
	)
	deleteW := httptest.NewRecorder()
	h.DeletePhoto(deleteW, deleteReq)
	if deleteW.Code != http.StatusOK {
		t.Fatalf("delete: status = %d, want 200; body = %s", deleteW.Code, deleteW.Body.String())
	}

	// Get after delete — should be 404
	getReq := httptest.NewRequest("GET", "/user/photos/headshot", nil)
	getReq = getReq.WithContext(middleware.WithFirebaseUID(getReq.Context(), firebaseUID))
	getW := httptest.NewRecorder()
	h.GetHeadshot(getW, getReq)
	if getW.Code != http.StatusNotFound {
		t.Fatalf("get after delete: status = %d, want 404; body = %s", getW.Code, getW.Body.String())
	}
}
