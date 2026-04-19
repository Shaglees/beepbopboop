package repository_test

import (
	"fmt"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func float64Ptr(f float64) *float64 { return &f }

func TestLocalCreatorRepo_UpsertAndListNearby(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewLocalCreatorRepo(db)

	lat, lon := 40.7128, -74.0060
	got, err := repo.Upsert(model.CreateCreatorRequest{
		Name:        "Maria Chen",
		Designation: "Painter",
		Bio:         "Brooklyn-based oil painter.",
		Lat:         float64Ptr(lat),
		Lon:         float64Ptr(lon),
		AreaName:    "Brooklyn, NY",
		Source:      "Brooklyn Rail",
	})
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
	if got.ID == "" {
		t.Error("expected non-empty ID after upsert")
	}
	if got.Name != "Maria Chen" {
		t.Errorf("expected name Maria Chen, got %s", got.Name)
	}

	creators, usedRadius, err := repo.ListNearby(lat, lon, 25.0, 20)
	if err != nil {
		t.Fatalf("ListNearby failed: %v", err)
	}
	if len(creators) != 1 {
		t.Errorf("expected 1 creator, got %d", len(creators))
	}
	if creators[0].Name != "Maria Chen" {
		t.Errorf("expected Maria Chen, got %s", creators[0].Name)
	}
	if usedRadius <= 0 {
		t.Error("expected positive usedRadius")
	}
}

func TestLocalCreatorRepo_Upsert_Idempotent(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewLocalCreatorRepo(db)

	lat, lon := 40.7128, -74.0060
	req := model.CreateCreatorRequest{
		Name:        "Jane Doe",
		Designation: "Sculptor",
		Lat:         float64Ptr(lat),
		Lon:         float64Ptr(lon),
		Source:      "test",
	}

	first, err := repo.Upsert(req)
	if err != nil {
		t.Fatalf("first upsert failed: %v", err)
	}

	req.Bio = "Updated bio"
	second, err := repo.Upsert(req)
	if err != nil {
		t.Fatalf("second upsert failed: %v", err)
	}

	if first.ID != second.ID {
		t.Errorf("expected same ID on upsert, got %s vs %s", first.ID, second.ID)
	}
	if second.Bio != "Updated bio" {
		t.Errorf("expected updated bio, got %q", second.Bio)
	}
}

func TestLocalCreatorRepo_ListNearby_AdaptiveRadius(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewLocalCreatorRepo(db)

	baseLat, baseLon := 40.7128, -74.0060

	// 5 creators within 1km
	for i := 0; i < 5; i++ {
		_, err := repo.Upsert(model.CreateCreatorRequest{
			Name:        fmt.Sprintf("Near Creator %d", i),
			Designation: "Painter",
			Lat:         float64Ptr(baseLat + float64(i)*0.001),
			Lon:         float64Ptr(baseLon),
			Source:      "test",
		})
		if err != nil {
			t.Fatalf("upsert near %d: %v", i, err)
		}
	}

	// 10 more creators ~30km away (lat +0.27 ≈ 30km)
	for i := 0; i < 10; i++ {
		_, err := repo.Upsert(model.CreateCreatorRequest{
			Name:        fmt.Sprintf("Far Creator %d", i),
			Designation: "Musician",
			Lat:         float64Ptr(baseLat + 0.27 + float64(i)*0.001),
			Lon:         float64Ptr(baseLon),
			Source:      "test",
		})
		if err != nil {
			t.Fatalf("upsert far %d: %v", i, err)
		}
	}

	// Starting radius 5km — only 5 near results, below threshold of 10.
	// Should expand until it finds ≥10 results.
	creators, usedRadius, err := repo.ListNearby(baseLat, baseLon, 5.0, 30)
	if err != nil {
		t.Fatalf("ListNearby: %v", err)
	}
	if usedRadius <= 5.0 {
		t.Errorf("expected radius to expand beyond 5km, got %.1f", usedRadius)
	}
	if len(creators) < 10 {
		t.Errorf("expected ≥10 creators after radius expansion, got %d", len(creators))
	}
}

func TestLocalCreatorRepo_ListNearby_OutOfRange(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewLocalCreatorRepo(db)

	// Creator in Brooklyn
	lat, lon := 40.7128, -74.0060
	_, err := repo.Upsert(model.CreateCreatorRequest{
		Name:        "Local Artist",
		Designation: "Painter",
		Lat:         float64Ptr(lat),
		Lon:         float64Ptr(lon),
		Source:      "test",
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// Query from London — should return zero results even after expansion
	creators, _, err := repo.ListNearby(51.5074, -0.1278, 25.0, 20)
	if err != nil {
		t.Fatalf("ListNearby: %v", err)
	}
	if len(creators) != 0 {
		t.Errorf("expected 0 creators from London, got %d", len(creators))
	}
}
