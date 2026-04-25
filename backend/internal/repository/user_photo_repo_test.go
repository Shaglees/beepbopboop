package repository_test

import (
	"bytes"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestUserPhotoRepo_SaveAndGetHeadshot(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	photoRepo := repository.NewUserPhotoRepo(db)

	user, err := userRepo.FindOrCreateByFirebaseUID("firebase-photo-headshot")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	fakeJPEG := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}
	contentType := "image/jpeg"

	if err := photoRepo.SaveHeadshot(user.ID, fakeJPEG, contentType); err != nil {
		t.Fatalf("SaveHeadshot: %v", err)
	}

	gotData, gotType, err := photoRepo.GetHeadshot(user.ID)
	if err != nil {
		t.Fatalf("GetHeadshot: %v", err)
	}
	if !bytes.Equal(gotData, fakeJPEG) {
		t.Errorf("headshot data mismatch: got %v, want %v", gotData, fakeJPEG)
	}
	if gotType != contentType {
		t.Errorf("headshot content type = %q, want %q", gotType, contentType)
	}
}

func TestUserPhotoRepo_SaveAndGetBodyshot(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	photoRepo := repository.NewUserPhotoRepo(db)

	user, err := userRepo.FindOrCreateByFirebaseUID("firebase-photo-bodyshot")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	fakePNG := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	contentType := "image/png"

	if err := photoRepo.SaveBodyshot(user.ID, fakePNG, contentType); err != nil {
		t.Fatalf("SaveBodyshot: %v", err)
	}

	gotData, gotType, err := photoRepo.GetBodyshot(user.ID)
	if err != nil {
		t.Fatalf("GetBodyshot: %v", err)
	}
	if !bytes.Equal(gotData, fakePNG) {
		t.Errorf("bodyshot data mismatch: got %v, want %v", gotData, fakePNG)
	}
	if gotType != contentType {
		t.Errorf("bodyshot content type = %q, want %q", gotType, contentType)
	}
}

func TestUserPhotoRepo_Delete(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	photoRepo := repository.NewUserPhotoRepo(db)

	user, err := userRepo.FindOrCreateByFirebaseUID("firebase-photo-delete")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	fakeJPEG := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}
	if err := photoRepo.SaveHeadshot(user.ID, fakeJPEG, "image/jpeg"); err != nil {
		t.Fatalf("SaveHeadshot: %v", err)
	}

	if err := photoRepo.DeletePhoto(user.ID, "headshot"); err != nil {
		t.Fatalf("DeletePhoto: %v", err)
	}

	gotData, gotType, err := photoRepo.GetHeadshot(user.ID)
	if err != nil {
		t.Fatalf("GetHeadshot after delete: %v", err)
	}
	if gotData != nil {
		t.Errorf("expected nil data after delete, got %v", gotData)
	}
	if gotType != "" {
		t.Errorf("expected empty content type after delete, got %q", gotType)
	}
}

func TestUserPhotoRepo_GetEmpty(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	photoRepo := repository.NewUserPhotoRepo(db)

	user, err := userRepo.FindOrCreateByFirebaseUID("firebase-photo-empty")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	gotData, gotType, err := photoRepo.GetHeadshot(user.ID)
	if err != nil {
		t.Fatalf("GetHeadshot on empty user: %v", err)
	}
	if gotData != nil {
		t.Errorf("expected nil data for user with no photo, got %v", gotData)
	}
	if gotType != "" {
		t.Errorf("expected empty content type for user with no photo, got %q", gotType)
	}
}
