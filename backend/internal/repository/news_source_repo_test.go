package repository_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// dublinSource returns a test NewsSource centred on Dublin, Ireland.
func dublinSource() model.NewsSource {
	return model.NewsSource{
		Name:        "Dublin Times",
		URL:         "https://dublintimes.ie",
		FeedURL:     "https://dublintimes.ie/rss",
		AreaLabel:   "Dublin",
		Latitude:    53.3498,
		Longitude:   -6.2603,
		RadiusKm:    30.0,
		Topics:      []string{"local", "politics"},
		TrustScore:  80,
		FetchMethod: "rss",
		Active:      true,
	}
}

func TestNewsSourceRepo_CreateAndList(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewNewsSourceRepo(db)

	src := dublinSource()
	if err := repo.Create(src); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Query from Dublin (should find the source — distance ~0 km, well within radius).
	dublinLat, dublinLon := 53.3498, -6.2603
	results, err := repo.List(dublinLat, dublinLon, 25.0, nil)
	if err != nil {
		t.Fatalf("List (Dublin): %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 result near Dublin, got 0")
	}
	found := false
	for _, r := range results {
		if r.URL == src.URL {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Dublin Times not found in results near Dublin")
	}

	// Query from New York City (should NOT find the Dublin source).
	nycLat, nycLon := 40.7128, -74.0060
	nycResults, err := repo.List(nycLat, nycLon, 25.0, nil)
	if err != nil {
		t.Fatalf("List (NYC): %v", err)
	}
	for _, r := range nycResults {
		if r.URL == src.URL {
			t.Errorf("Dublin Times should NOT appear in NYC results")
		}
	}
}

func TestNewsSourceRepo_ListByTopics(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewNewsSourceRepo(db)

	sports := model.NewsSource{
		Name:        "Dublin Sports Daily",
		URL:         "https://dublinsports.ie",
		AreaLabel:   "Dublin",
		Latitude:    53.3498,
		Longitude:   -6.2603,
		RadiusKm:    30.0,
		Topics:      []string{"sports"},
		TrustScore:  60,
		FetchMethod: "rss",
		Active:      true,
	}
	politics := model.NewsSource{
		Name:        "Dublin Politics",
		URL:         "https://dublinpolitics.ie",
		AreaLabel:   "Dublin",
		Latitude:    53.3498,
		Longitude:   -6.2603,
		RadiusKm:    30.0,
		Topics:      []string{"politics"},
		TrustScore:  70,
		FetchMethod: "rss",
		Active:      true,
	}

	if err := repo.Create(sports); err != nil {
		t.Fatalf("Create sports: %v", err)
	}
	if err := repo.Create(politics); err != nil {
		t.Fatalf("Create politics: %v", err)
	}

	// Filter by "sports" topic — should only see sports source.
	results, err := repo.List(53.3498, -6.2603, 50.0, []string{"sports"})
	if err != nil {
		t.Fatalf("List by sports: %v", err)
	}

	foundSports := false
	foundPolitics := false
	for _, r := range results {
		if r.URL == sports.URL {
			foundSports = true
		}
		if r.URL == politics.URL {
			foundPolitics = true
		}
	}
	if !foundSports {
		t.Error("expected sports source when filtering by 'sports'")
	}
	if foundPolitics {
		t.Error("politics source should not appear when filtering by 'sports'")
	}
}

func TestNewsSourceRepo_Get(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewNewsSourceRepo(db)

	src := model.NewsSource{
		Name:        "Cork Examiner",
		URL:         "https://corkexaminer.ie",
		AreaLabel:   "Cork",
		Latitude:    51.8985,
		Longitude:   -8.4756,
		RadiusKm:    20.0,
		Topics:      []string{"local"},
		TrustScore:  75,
		FetchMethod: "rss",
		Active:      true,
	}
	if err := repo.Create(src); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Retrieve the ID via List so we can call Get.
	results, err := repo.List(51.8985, -8.4756, 5.0, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var id string
	for _, r := range results {
		if r.URL == src.URL {
			id = r.ID
			break
		}
	}
	if id == "" {
		t.Fatal("could not find Cork Examiner ID after create")
	}

	got, err := repo.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil, expected a result")
	}
	if got.URL != src.URL {
		t.Errorf("Get URL = %q, want %q", got.URL, src.URL)
	}
	if got.Name != src.Name {
		t.Errorf("Get Name = %q, want %q", got.Name, src.Name)
	}

	// Get a non-existent ID should return nil.
	missing, err := repo.Get("00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("Get missing: %v", err)
	}
	if missing != nil {
		t.Errorf("Get missing expected nil, got %+v", missing)
	}
}

func TestNewsSourceRepo_Upsert(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewNewsSourceRepo(db)

	src := model.NewsSource{
		Name:        "Galway Tribune",
		URL:         "https://galwaytribune.ie",
		AreaLabel:   "Galway",
		Latitude:    53.2707,
		Longitude:   -9.0568,
		RadiusKm:    25.0,
		Topics:      []string{"local"},
		TrustScore:  55,
		FetchMethod: "rss",
		Active:      true,
	}

	// First create.
	if err := repo.Create(src); err != nil {
		t.Fatalf("Create (first): %v", err)
	}

	// Second create with same URL but different TrustScore — should update, not duplicate.
	src.TrustScore = 90
	src.Name = "Galway Tribune Updated"
	if err := repo.Create(src); err != nil {
		t.Fatalf("Create (second/upsert): %v", err)
	}

	// Verify there is only one row and it has the updated values.
	results, err := repo.List(53.2707, -9.0568, 5.0, nil)
	if err != nil {
		t.Fatalf("List after upsert: %v", err)
	}

	count := 0
	for _, r := range results {
		if r.URL == src.URL {
			count++
			if r.TrustScore != 90 {
				t.Errorf("TrustScore after upsert = %d, want 90", r.TrustScore)
			}
			if r.Name != "Galway Tribune Updated" {
				t.Errorf("Name after upsert = %q, want %q", r.Name, "Galway Tribune Updated")
			}
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 row for URL %q after upsert, got %d", src.URL, count)
	}
}
