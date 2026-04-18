package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestPostHandler_CreatePost(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"title": "Tennis courts nearby", "body": "A park near you has tennis courts."}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["title"] != "Tennis courts nearby" {
		t.Errorf("expected title, got %v", resp["title"])
	}
	if resp["agent_name"] != "My Agent" {
		t.Errorf("expected agent_name My Agent, got %v", resp["agent_name"])
	}
}

func TestPostHandler_CreatePost_DefaultPostType(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"title": "A nice park", "body": "Great park nearby."}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["post_type"] != "discovery" {
		t.Errorf("expected post_type discovery, got %v", resp["post_type"])
	}
}

func TestPostHandler_CreatePost_InvalidPostType(t *testing.T) {
	db := database.OpenTestDB(t)

	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"title": "Test", "body": "Test body", "post_type": "bogus"}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), "some-agent"))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPostHandler_CreatePost_ArticleType(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"title": "New AI breakthrough", "body": "A major advance in reasoning.", "post_type": "article", "locality": "Anthropic Blog"}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["post_type"] != "article" {
		t.Errorf("expected post_type article, got %v", resp["post_type"])
	}
}

func TestPostHandler_CreatePost_VideoType(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"title": "WebGPU explainer", "body": "A 12-minute deep dive.", "post_type": "video", "locality": "Fireship on YouTube"}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["post_type"] != "video" {
		t.Errorf("expected post_type video, got %v", resp["post_type"])
	}
}

func TestPostHandler_DefaultVisibility(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"title": "Test post", "body": "Test body"}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["visibility"] != "public" {
		t.Errorf("expected visibility public, got %v", resp["visibility"])
	}
}

func TestPostHandler_PersonalVisibility(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"title": "Personal post", "body": "Family stuff", "visibility": "personal"}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["visibility"] != "personal" {
		t.Errorf("expected visibility personal, got %v", resp["visibility"])
	}
}

func TestPostHandler_PrivateVisibility(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"title": "Private post", "body": "Calendar event", "visibility": "private"}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["visibility"] != "private" {
		t.Errorf("expected visibility private, got %v", resp["visibility"])
	}
}

func TestPostHandler_InvalidVisibility(t *testing.T) {
	db := database.OpenTestDB(t)

	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"title": "Test", "body": "Test body", "visibility": "secret"}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), "some-agent"))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPostHandler_LabelsRoundTrip(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"title": "Labeled post", "body": "Post with labels", "labels": ["coffee", "cafe", "specialty-coffee"]}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)

	labels, ok := resp["labels"].([]any)
	if !ok {
		t.Fatalf("expected labels array, got %T: %v", resp["labels"], resp["labels"])
	}
	if len(labels) != 3 {
		t.Errorf("expected 3 labels, got %d", len(labels))
	}
	if labels[0] != "coffee" || labels[1] != "cafe" || labels[2] != "specialty-coffee" {
		t.Errorf("unexpected labels: %v", labels)
	}
}

func TestPostHandler_OutfitSingleHero(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	// Template 1A: single hero image — full-width 16:10 landscape
	body := `{
		"title": "The Linen Shirt Renaissance",
		"body": "Relaxed linen is back for summer.\n\n**Trend:** Linen everything\n**For you:** Go slightly oversized in a camp collar.\n**Try:** Sunspel Linen Camp Collar ($195)\n**Alt:** Uniqlo Premium Linen ($39)",
		"display_hint": "outfit",
		"post_type": "article",
		"visibility": "personal",
		"labels": ["fashion", "outfit", "linen", "summer"],
		"images": [
			{"url": "https://example.com/linen-hero.jpg", "role": "hero"}
		]
	}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)

	images, ok := resp["images"].([]any)
	if !ok {
		t.Fatalf("expected images array, got %T: %v", resp["images"], resp["images"])
	}
	if len(images) != 1 {
		t.Errorf("expected 1 image, got %d", len(images))
	}
	hero := images[0].(map[string]any)
	if hero["role"] != "hero" {
		t.Errorf("expected role hero, got %v", hero["role"])
	}
	if resp["display_hint"] != "outfit" {
		t.Errorf("expected display_hint outfit, got %v", resp["display_hint"])
	}
}

func TestPostHandler_OutfitHeroAndDetail(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	// Templates 2A/2B/2C: hero + detail — iOS selects variant by hash(post.id) % 3
	body := `{
		"title": "The Unstructured Blazer is Having a Moment",
		"body": "Deconstructed blazers are filtering into everyday wear.\n\n**Trend:** The unstructured blazer\n**For you:** Go slightly cropped with a wider shoulder.\n**Try:** COS Deconstructed Blazer ($175) · A.P.C. Cotton Blazer ($220)\n**Alt:** Zara Soft Blazer ($89)",
		"display_hint": "outfit",
		"post_type": "article",
		"visibility": "personal",
		"labels": ["fashion", "outfit", "blazers", "smart-casual", "spring"],
		"images": [
			{"url": "https://example.com/blazer-hero.jpg", "role": "hero"},
			{"url": "https://example.com/blazer-detail.jpg", "role": "detail", "caption": "Styling close-up"}
		]
	}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)

	images, ok := resp["images"].([]any)
	if !ok {
		t.Fatalf("expected images array, got %T: %v", resp["images"], resp["images"])
	}
	if len(images) != 2 {
		t.Errorf("expected 2 images, got %d", len(images))
	}

	roles := make(map[string]bool)
	for _, img := range images {
		m := img.(map[string]any)
		roles[m["role"].(string)] = true
	}
	if !roles["hero"] || !roles["detail"] {
		t.Errorf("expected hero and detail roles, got %v", roles)
	}
}

func TestPostHandler_OutfitFullWithProducts(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	// Full outfit: hero + details + products — covers collage + product thumbnail row
	body := `{
		"title": "Wide-Leg Trousers Are the Move This Season",
		"body": "The wide-leg trouser is replacing slim-fit everywhere.\n\n**Trend:** Wide-leg trousers\n**For you:** High-waisted with a slight taper at the ankle works best.\n**Try:** Lemaire Wide Trouser ($490) · AMI Paris Pleated ($320) · COS Elastic Waist ($115)\n**Alt:** Weekday Uno Wide ($59)",
		"display_hint": "outfit",
		"image_url": "https://example.com/trousers-hero.jpg",
		"post_type": "article",
		"visibility": "personal",
		"labels": ["fashion", "outfit", "trousers", "wide-leg", "spring"],
		"images": [
			{"url": "https://example.com/trousers-hero.jpg", "role": "hero"},
			{"url": "https://example.com/trousers-detail1.jpg", "role": "detail"},
			{"url": "https://example.com/trousers-detail2.jpg", "role": "detail", "caption": "Pleat detail"},
			{"url": "https://example.com/lemaire-product.jpg", "role": "product", "caption": "Lemaire"},
			{"url": "https://example.com/ami-product.jpg", "role": "product", "caption": "AMI Paris"},
			{"url": "https://example.com/cos-product.jpg", "role": "product", "caption": "COS"}
		]
	}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)

	images, ok := resp["images"].([]any)
	if !ok {
		t.Fatalf("expected images array, got %T: %v", resp["images"], resp["images"])
	}
	if len(images) != 6 {
		t.Errorf("expected 6 images, got %d", len(images))
	}

	// Count by role
	roleCounts := make(map[string]int)
	for _, img := range images {
		m := img.(map[string]any)
		roleCounts[m["role"].(string)]++
	}
	if roleCounts["hero"] != 1 {
		t.Errorf("expected 1 hero, got %d", roleCounts["hero"])
	}
	if roleCounts["detail"] != 2 {
		t.Errorf("expected 2 detail, got %d", roleCounts["detail"])
	}
	if roleCounts["product"] != 3 {
		t.Errorf("expected 3 product, got %d", roleCounts["product"])
	}

	// Verify product captions round-trip
	var productCaptions []string
	for _, img := range images {
		m := img.(map[string]any)
		if m["role"] == "product" {
			if c, ok := m["caption"].(string); ok {
				productCaptions = append(productCaptions, c)
			}
		}
	}
	if len(productCaptions) != 3 {
		t.Errorf("expected 3 product captions, got %d: %v", len(productCaptions), productCaptions)
	}
}

func TestPostHandler_ImagesNullOmitted(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	// Post without images — images should be omitted (nil) in response
	body := `{"title": "No images post", "body": "Just text content."}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)

	if _, exists := resp["images"]; exists {
		t.Errorf("expected images to be omitted for post without images, got %v", resp["images"])
	}
}

func TestPostHandler_OutfitProductsOnly(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	// Products only, no hero/detail — tests fallback to image_url for collage
	body := `{
		"title": "Summer Sneaker Rotation",
		"body": "Three pairs to rotate all summer.\n\n**Trend:** Retro runners\n**For you:** Stick to neutral tones with one pop colour pair.\n**Try:** New Balance 990v6 ($185) · Salomon XT-6 ($190) · Adidas Samba ($100)\n**Alt:** Puma Palermo ($80)",
		"display_hint": "outfit",
		"image_url": "https://example.com/sneakers-main.jpg",
		"post_type": "article",
		"visibility": "personal",
		"labels": ["fashion", "outfit", "sneakers", "summer"],
		"images": [
			{"url": "https://example.com/nb990-product.jpg", "role": "product", "caption": "New Balance 990v6"},
			{"url": "https://example.com/salomon-product.jpg", "role": "product", "caption": "Salomon XT-6"},
			{"url": "https://example.com/samba-product.jpg", "role": "product", "caption": "Adidas Samba"}
		]
	}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)

	images, ok := resp["images"].([]any)
	if !ok {
		t.Fatalf("expected images array, got %T: %v", resp["images"], resp["images"])
	}
	if len(images) != 3 {
		t.Errorf("expected 3 images, got %d", len(images))
	}
	for _, img := range images {
		m := img.(map[string]any)
		if m["role"] != "product" {
			t.Errorf("expected all product roles, got %v", m["role"])
		}
	}

	// image_url should still be set for backwards compat
	if resp["image_url"] != "https://example.com/sneakers-main.jpg" {
		t.Errorf("expected image_url preserved, got %v", resp["image_url"])
	}
}

func TestPostHandler_MissingTitle(t *testing.T) {
	db := database.OpenTestDB(t)

	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"body": "no title"}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), "some-agent"))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// ============================================================================
// Lint endpoint tests
// ============================================================================

// lintCall is a helper that POSTs to /posts/lint and returns the parsed response.
func lintCall(t *testing.T, h *handler.PostHandler, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest("POST", "/posts/lint", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), "lint-agent"))
	rec := httptest.NewRecorder()
	h.LintPost(rec, req)
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	return rec.Code, resp
}

func lintErrors(resp map[string]any) []any {
	if e, ok := resp["errors"].([]any); ok {
		return e
	}
	return nil
}

func lintWarnings(resp map[string]any) []any {
	if w, ok := resp["warnings"].([]any); ok {
		return w
	}
	return nil
}

func hasFieldError(issues []any, field string) bool {
	for _, i := range issues {
		m := i.(map[string]any)
		if m["field"] == field {
			return true
		}
	}
	return false
}

func TestLintPost_MissingTitleAndBody(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	code, resp := lintCall(t, h, `{}`)
	if code != 200 {
		t.Fatalf("expected 200, got %d", code)
	}
	if resp["valid"] != false {
		t.Error("expected valid=false")
	}
	errs := lintErrors(resp)
	if !hasFieldError(errs, "title") {
		t.Error("expected error for missing title")
	}
	if !hasFieldError(errs, "body") {
		t.Error("expected error for missing body")
	}
}

func TestLintPost_ValidMinimal(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	code, resp := lintCall(t, h, `{"title":"test","body":"hello"}`)
	if code != 200 {
		t.Fatalf("expected 200, got %d", code)
	}
	if resp["valid"] != true {
		t.Errorf("expected valid=true, got errors: %v", lintErrors(resp))
	}
	// Should still have warnings (no labels, no locality)
	warns := lintWarnings(resp)
	if !hasFieldError(warns, "labels") {
		t.Error("expected warning for missing labels")
	}
}

func TestLintPost_InvalidPostType(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	_, resp := lintCall(t, h, `{"title":"t","body":"b","post_type":"bogus"}`)
	if resp["valid"] != false {
		t.Error("expected valid=false for bad post_type")
	}
	if !hasFieldError(lintErrors(resp), "post_type") {
		t.Error("expected error for invalid post_type")
	}
}

func TestLintPost_InvalidVisibility(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	_, resp := lintCall(t, h, `{"title":"t","body":"b","visibility":"secret"}`)
	if resp["valid"] != false {
		t.Error("expected valid=false")
	}
	if !hasFieldError(lintErrors(resp), "visibility") {
		t.Error("expected error for invalid visibility")
	}
}

func TestLintPost_InvalidDisplayHint(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	_, resp := lintCall(t, h, `{"title":"t","body":"b","display_hint":"foobar"}`)
	if resp["valid"] != false {
		t.Error("expected valid=false for unknown display_hint")
	}
	if !hasFieldError(lintErrors(resp), "display_hint") {
		t.Error("expected error for invalid display_hint")
	}
}

func TestLintPost_ScoreboardMissingExternalURL(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	_, resp := lintCall(t, h, `{"title":"t","body":"b","display_hint":"scoreboard"}`)
	if resp["valid"] != false {
		t.Error("expected valid=false")
	}
	if !hasFieldError(lintErrors(resp), "external_url") {
		t.Error("expected error for missing external_url on scoreboard")
	}
}

func TestLintPost_ScoreboardBadJSON(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	_, resp := lintCall(t, h, `{"title":"t","body":"b","display_hint":"scoreboard","external_url":"{}"}`)
	if resp["valid"] != false {
		t.Error("expected valid=false for empty scoreboard data")
	}
	errs := lintErrors(resp)
	if !hasFieldError(errs, "external_url.status") {
		t.Error("expected error for missing status")
	}
	if !hasFieldError(errs, "external_url.home") {
		t.Error("expected error for missing home")
	}
	if !hasFieldError(errs, "external_url.away") {
		t.Error("expected error for missing away")
	}
}

func TestLintPost_ScoreboardValid(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	gameJSON := `{"status":"Final","home":{"name":"Lakers","abbr":"LAL","score":110},"away":{"name":"Celtics","abbr":"BOS","score":105},"sport":"NBA"}`
	body := `{"title":"t","body":"b","display_hint":"scoreboard","external_url":` + jsonString(gameJSON) + `,"labels":["nba"]}`
	_, resp := lintCall(t, h, body)
	if resp["valid"] != true {
		t.Errorf("expected valid=true for good scoreboard, errors: %v", lintErrors(resp))
	}
}

func TestLintPost_MatchupMissingGameTimeWarning(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	gameJSON := `{"status":"Scheduled","home":{"name":"Lakers","abbr":"LAL"},"away":{"name":"Celtics","abbr":"BOS"}}`
	body := `{"title":"t","body":"b","display_hint":"matchup","external_url":` + jsonString(gameJSON) + `,"labels":["nba"]}`
	_, resp := lintCall(t, h, body)
	if resp["valid"] != true {
		t.Errorf("expected valid=true, errors: %v", lintErrors(resp))
	}
	warns := lintWarnings(resp)
	if !hasFieldError(warns, "external_url.gameTime") {
		t.Error("expected warning for missing gameTime on matchup")
	}
	if !hasFieldError(warns, "external_url.sport") {
		t.Error("expected warning for missing sport")
	}
}

func TestLintPost_WeatherMissingFields(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	// Weather with empty current object — should flag all required fields
	weatherJSON := `{"current":{},"location":{}}`
	body := `{"title":"t","body":"b","display_hint":"weather","external_url":` + jsonString(weatherJSON) + `,"labels":["weather"]}`
	_, resp := lintCall(t, h, body)
	if resp["valid"] != false {
		t.Error("expected valid=false for incomplete weather data")
	}
	errs := lintErrors(resp)
	if !hasFieldError(errs, "external_url.current.temp_c") {
		t.Error("expected error for missing temp_c")
	}
	if !hasFieldError(errs, "external_url.hourly") {
		t.Error("expected error for missing hourly")
	}
	if !hasFieldError(errs, "external_url.daily") {
		t.Error("expected error for missing daily")
	}
	if !hasFieldError(errs, "external_url.location.latitude") {
		t.Error("expected error for missing location.latitude")
	}
}

func TestLintPost_StandingsValid(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	standingsJSON := `{"league":"NBA","date":"2026-04-16","games":[{"home":"LAL","away":"BOS","homeScore":110,"awayScore":105,"status":"Final"}]}`
	body := `{"title":"t","body":"b","display_hint":"standings","external_url":` + jsonString(standingsJSON) + `,"labels":["nba"]}`
	_, resp := lintCall(t, h, body)
	if resp["valid"] != true {
		t.Errorf("expected valid=true for good standings, errors: %v", lintErrors(resp))
	}
}

func TestLintPost_StandingsMissingGameFields(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	standingsJSON := `{"league":"NBA","date":"2026-04-16","games":[{}]}`
	body := `{"title":"t","body":"b","display_hint":"standings","external_url":` + jsonString(standingsJSON) + `,"labels":["nba"]}`
	_, resp := lintCall(t, h, body)
	if resp["valid"] != false {
		t.Error("expected valid=false for standings with empty game")
	}
	errs := lintErrors(resp)
	if !hasFieldError(errs, "external_url.games[0].home") {
		t.Error("expected error for missing home in game 0")
	}
	if !hasFieldError(errs, "external_url.games[0].status") {
		t.Error("expected error for missing status in game 0")
	}
}

func TestLintPost_ImagesNoURL(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	body := `{"title":"t","body":"b","images":[{"role":"hero"}],"labels":["test"]}`
	_, resp := lintCall(t, h, body)
	if resp["valid"] != false {
		t.Error("expected valid=false for image without url")
	}
	if !hasFieldError(lintErrors(resp), "images[0].url") {
		t.Error("expected error for missing image url")
	}
}

func TestLintPost_UnknownImageRoleWarning(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	body := `{"title":"t","body":"b","images":[{"url":"https://example.com/img.jpg","role":"thumbnail"}],"labels":["test"]}`
	_, resp := lintCall(t, h, body)
	if resp["valid"] != true {
		t.Errorf("unknown image role should be warning not error, errors: %v", lintErrors(resp))
	}
	if !hasFieldError(lintWarnings(resp), "images[0].role") {
		t.Error("expected warning for unknown image role")
	}
}

func TestLintPost_LabelsWarningTooMany(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	// 21 labels
	labels := `["a","b","c","d","e","f","g","h","i","j","k","l","m","n","o","p","q","r","s","t","u"]`
	body := `{"title":"t","body":"b","labels":` + labels + `}`
	_, resp := lintCall(t, h, body)
	if resp["valid"] != true {
		t.Errorf("too many labels should be warning not error, errors: %v", lintErrors(resp))
	}
	if !hasFieldError(lintWarnings(resp), "labels") {
		t.Error("expected warning for >20 labels")
	}
}

func TestLintPost_CreatePostRejectsInvalid(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	// Scoreboard with no external_url should be rejected with structured errors
	body := `{"title":"t","body":"b","display_hint":"scoreboard"}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), "some-agent"))
	rec := httptest.NewRecorder()
	h.CreatePost(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["valid"] != false {
		t.Error("expected valid=false in rejection response")
	}
	if !hasFieldError(lintErrors(resp), "external_url") {
		t.Error("expected structured error for missing external_url")
	}
}

// ============================================================================
// Sync tests — ensure the lookup maps stay in sync with what tests cover
// ============================================================================

// TestValidPostTypes_AllTestedViaLint verifies every value in ValidPostTypes
// is accepted by lint (and that invalid values are rejected).
func TestValidPostTypes_AllTestedViaLint(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	for pt := range handler.ValidPostTypes {
		body := `{"title":"t","body":"b","post_type":"` + pt + `","labels":["sync-test"]}`
		_, resp := lintCall(t, h, body)
		if resp["valid"] != true {
			t.Errorf("ValidPostTypes contains %q but lint rejects it: %v", pt, lintErrors(resp))
		}
	}

	// Confirm an unknown value is rejected
	body := `{"title":"t","body":"b","post_type":"unknown_type","labels":["sync-test"]}`
	_, resp := lintCall(t, h, body)
	if resp["valid"] != false {
		t.Error("expected invalid post_type to be rejected")
	}
}

// TestValidVisibility_AllTestedViaLint verifies every value in ValidVisibility
// is accepted by lint.
func TestValidVisibility_AllTestedViaLint(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	for vis := range handler.ValidVisibility {
		body := `{"title":"t","body":"b","visibility":"` + vis + `","labels":["sync-test"]}`
		_, resp := lintCall(t, h, body)
		if resp["valid"] != true {
			t.Errorf("ValidVisibility contains %q but lint rejects it: %v", vis, lintErrors(resp))
		}
	}
}

func TestLintPost_EntertainmentMissingExternalURL(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	_, resp := lintCall(t, h, `{"title":"t","body":"b","display_hint":"entertainment"}`)
	if resp["valid"] != false {
		t.Error("expected valid=false when entertainment has no external_url")
	}
	if !hasFieldError(lintErrors(resp), "external_url") {
		t.Error("expected error for missing external_url on entertainment")
	}
}

func TestLintPost_EntertainmentBadJSON(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	_, resp := lintCall(t, h, `{"title":"t","body":"b","display_hint":"entertainment","external_url":"{}"}`)
	if resp["valid"] != false {
		t.Error("expected valid=false for empty entertainment data")
	}
	errs := lintErrors(resp)
	if !hasFieldError(errs, "external_url.subject") {
		t.Error("expected error for missing subject")
	}
	if !hasFieldError(errs, "external_url.headline") {
		t.Error("expected error for missing headline")
	}
	if !hasFieldError(errs, "external_url.source") {
		t.Error("expected error for missing source")
	}
}

func TestLintPost_EntertainmentValid(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	entJSON := `{"subject":"Zendaya","headline":"Zendaya Named TIME Entertainer of the Year","source":"People","category":"award","tags":["awards","zendaya"]}`
	body := `{"title":"t","body":"b","display_hint":"entertainment","external_url":` + jsonString(entJSON) + `,"labels":["entertainment"]}`
	_, resp := lintCall(t, h, body)
	if resp["valid"] != true {
		t.Errorf("expected valid=true for good entertainment data, errors: %v", lintErrors(resp))
	}
}

// TestValidDisplayHints_AllTestedViaLint verifies every value in ValidDisplayHints
// is accepted by lint (without requiring external_url for non-structured hints).
func TestValidDisplayHints_AllTestedViaLint(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	// Hints that need structured external_url
	structuredHints := map[string]string{
		"weather":    `{"current":{"temp_c":20,"feels_like_c":18,"humidity":60,"wind_speed_kmh":10,"uv_index":5,"is_day":true,"condition":"Sunny","condition_code":1000},"hourly":[],"daily":[],"location":{"latitude":53.3,"longitude":-6.2,"timezone":"Europe/Dublin"}}`,
		"scoreboard": `{"status":"Final","home":{"name":"Lakers","abbr":"LAL"},"away":{"name":"Celtics","abbr":"BOS"},"sport":"NBA"}`,
		"matchup":    `{"status":"Scheduled","home":{"name":"Lakers","abbr":"LAL"},"away":{"name":"Celtics","abbr":"BOS"},"sport":"NBA","gameTime":"2026-04-16T19:00:00Z"}`,
		"standings":     `{"league":"NBA","date":"2026-04-16","games":[{"home":"LAL","away":"BOS","homeScore":110,"awayScore":105,"status":"Final"}]}`,
		"entertainment": `{"subject":"Zendaya","headline":"Zendaya Named TIME Entertainer of the Year","source":"People","category":"award","tags":["entertainment"]}`,
	}

	for hint := range handler.ValidDisplayHints {
		var body string
		if extURL, ok := structuredHints[hint]; ok {
			body = `{"title":"t","body":"b","display_hint":"` + hint + `","external_url":` + jsonString(extURL) + `,"labels":["sync-test"]}`
		} else {
			body = `{"title":"t","body":"b","display_hint":"` + hint + `","labels":["sync-test"]}`
		}
		_, resp := lintCall(t, h, body)
		if resp["valid"] != true {
			t.Errorf("ValidDisplayHints contains %q but lint rejects it: %v", hint, lintErrors(resp))
		}
	}
}

// TestValidImageRoles_AllTestedViaLint verifies known image roles don't produce warnings.
func TestValidImageRoles_AllTestedViaLint(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	for role := range handler.ValidImageRoles {
		body := `{"title":"t","body":"b","images":[{"url":"https://example.com/img.jpg","role":"` + role + `"}],"labels":["sync-test"]}`
		_, resp := lintCall(t, h, body)
		if resp["valid"] != true {
			t.Errorf("ValidImageRoles contains %q but lint rejects it: %v", role, lintErrors(resp))
		}
		// Should NOT have a warning for this role
		for _, w := range lintWarnings(resp) {
			m := w.(map[string]any)
			if m["field"] == "images[0].role" {
				t.Errorf("ValidImageRoles contains %q but lint warns about it", role)
			}
		}
	}
}

// TestValidationMaps_Sorted ensures the maps have deterministic iteration
// by checking they contain the expected values (catches accidental deletions).
func TestValidationMaps_Sorted(t *testing.T) {
	expectedPostTypes := []string{"article", "discovery", "event", "place", "video"}
	expectedVisibility := []string{"personal", "private", "public"}
	expectedHints := []string{"article", "brief", "calendar", "card", "comparison", "deal", "digest", "entertainment", "event", "matchup", "outfit", "place", "scoreboard", "standings", "weather"}
	expectedRoles := []string{"detail", "hero", "product"}

	checkMap := func(name string, m map[string]bool, expected []string) {
		var keys []string
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		if len(keys) != len(expected) {
			t.Errorf("%s: expected %d entries, got %d: %v", name, len(expected), len(keys), keys)
			return
		}
		for i, k := range keys {
			if k != expected[i] {
				t.Errorf("%s[%d]: expected %q, got %q", name, i, expected[i], k)
			}
		}
	}

	checkMap("ValidPostTypes", handler.ValidPostTypes, expectedPostTypes)
	checkMap("ValidVisibility", handler.ValidVisibility, expectedVisibility)
	checkMap("ValidDisplayHints", handler.ValidDisplayHints, expectedHints)
	checkMap("ValidImageRoles", handler.ValidImageRoles, expectedRoles)
}

// ============================================================================
// URL validation tests (issue #12)
// ============================================================================

func TestLintPost_InvalidImageURL(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	tests := []struct {
		name     string
		imageURL string
		wantErr  bool
	}{
		{"valid https", "https://example.com/img.jpg", false},
		{"valid http", "http://example.com/img.jpg", false},
		{"ftp scheme", "ftp://example.com/img.jpg", true},
		{"no scheme", "example.com/img.jpg", true},
		{"javascript scheme", "javascript:alert(1)", true},
		{"data uri", "data:image/png;base64,abc", true},
		{"empty host", "https:///path", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := `{"title":"t","body":"b","image_url":` + jsonString(tt.imageURL) + `,"labels":["test"]}`
			_, resp := lintCall(t, h, body)
			hasErr := hasFieldError(lintErrors(resp), "image_url")
			if hasErr != tt.wantErr {
				t.Errorf("image_url=%q: wantErr=%v, gotErr=%v, errors=%v", tt.imageURL, tt.wantErr, hasErr, lintErrors(resp))
			}
		})
	}
}

func TestLintPost_InvalidExternalURL(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	tests := []struct {
		name        string
		externalURL string
		displayHint string
		wantErr     bool
	}{
		{"valid https", "https://example.com/article", "", false},
		{"ftp scheme", "ftp://example.com/file", "", true},
		{"no scheme", "not-a-url", "", true},
		// Structured hints store JSON, not URLs — should NOT be validated as URL
		{"scoreboard JSON ok", `{"status":"Final","home":{"name":"A","abbr":"A"},"away":{"name":"B","abbr":"B"}}`, "scoreboard", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body string
			if tt.displayHint != "" {
				body = `{"title":"t","body":"b","external_url":` + jsonString(tt.externalURL) + `,"display_hint":"` + tt.displayHint + `","labels":["test"]}`
			} else {
				body = `{"title":"t","body":"b","external_url":` + jsonString(tt.externalURL) + `,"labels":["test"]}`
			}
			_, resp := lintCall(t, h, body)
			hasErr := hasFieldError(lintErrors(resp), "external_url")
			if hasErr != tt.wantErr {
				t.Errorf("external_url=%q hint=%q: wantErr=%v, gotErr=%v, errors=%v",
					tt.externalURL, tt.displayHint, tt.wantErr, hasErr, lintErrors(resp))
			}
		})
	}
}

func TestLintPost_URLTooLong(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	longURL := "https://example.com/" + string(make([]byte, 2100))
	body := `{"title":"t","body":"b","image_url":` + jsonString(longURL) + `,"labels":["test"]}`
	_, resp := lintCall(t, h, body)
	if !hasFieldError(lintErrors(resp), "image_url") {
		t.Error("expected error for URL exceeding max length")
	}
}

func TestLintPost_ImageArrayURLValidation(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))

	// Valid URL in images array — should pass
	body := `{"title":"t","body":"b","images":[{"url":"https://example.com/img.jpg","role":"hero"}],"labels":["test"]}`
	_, resp := lintCall(t, h, body)
	if hasFieldError(lintErrors(resp), "images[0].url") {
		t.Errorf("valid image URL should not error, got: %v", lintErrors(resp))
	}

	// Invalid URL in images array — should fail
	body = `{"title":"t","body":"b","images":[{"url":"ftp://bad.com/img.jpg","role":"hero"}],"labels":["test"]}`
	_, resp = lintCall(t, h, body)
	if !hasFieldError(lintErrors(resp), "images[0].url") {
		t.Error("expected error for invalid image URL in array")
	}
}

// jsonString returns a JSON-encoded string value (with escaping).
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// ============================================================================
// Scheduling tests
// ============================================================================

func TestPostHandler_CreatePost_Scheduled(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-sched")
	agent, _ := agentRepo.Create(user.ID, "Sched Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	futureTime := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
	body := `{"title": "Future post", "body": "This should be scheduled.", "scheduled_at": "` + futureTime + `", "labels": ["test"]}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["status"] != "scheduled" {
		t.Errorf("expected status=scheduled, got %v", resp["status"])
	}
	if resp["scheduled_at"] == nil {
		t.Error("expected scheduled_at to be set")
	}
}

func TestPostHandler_CreatePost_ImmediateWhenNoSchedule(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-imm")
	agent, _ := agentRepo.Create(user.ID, "Imm Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"title": "Immediate post", "body": "Published right away.", "labels": ["test"]}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["status"] != "published" {
		t.Errorf("expected status=published, got %v", resp["status"])
	}
}

func TestPostHandler_ListPosts_StatusFilter(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-filter")
	agent, _ := agentRepo.Create(user.ID, "Filter Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	// Create one immediate and one scheduled post
	for _, b := range []string{
		`{"title": "Now post", "body": "Published.", "labels": ["test"]}`,
		`{"title": "Later post", "body": "Scheduled.", "scheduled_at": "` + time.Now().Add(2*time.Hour).UTC().Format(time.RFC3339) + `", "labels": ["test"]}`,
	} {
		req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(b))
		req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
		rec := httptest.NewRecorder()
		h.CreatePost(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create failed: %d %s", rec.Code, rec.Body.String())
		}
	}

	// List scheduled only
	req := httptest.NewRequest("GET", "/posts?status=scheduled", nil)
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()
	h.ListPosts(rec, req)

	var posts []map[string]any
	json.NewDecoder(rec.Body).Decode(&posts)
	if len(posts) != 1 {
		t.Fatalf("expected 1 scheduled post, got %d", len(posts))
	}
	if posts[0]["title"] != "Later post" {
		t.Errorf("expected 'Later post', got %v", posts[0]["title"])
	}
}

func TestPostRepo_PublishScheduled(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-pub")
	agent, _ := agentRepo.Create(user.ID, "Pub Agent")

	// Create a post scheduled in the past (should be published immediately by worker)
	pastTime := time.Now().Add(-1 * time.Minute)
	_, err := postRepo.Create(repository.CreatePostParams{
		AgentID:     agent.ID,
		UserID:      user.ID,
		Title:       "Past scheduled",
		Body:        "Should get published.",
		PostType:    "discovery",
		Visibility:  "public",
		Labels:      []string{"test"},
		ScheduledAt: &pastTime,
	})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}

	// The post has scheduled_at in the past, but Create sets status=published
	// when scheduled_at is not in the future, so let's manually create one with
	// a future scheduled_at, then manipulate it to test the worker.
	futureTime := time.Now().Add(2 * time.Hour)
	post2, err := postRepo.Create(repository.CreatePostParams{
		AgentID:     agent.ID,
		UserID:      user.ID,
		Title:       "Future scheduled",
		Body:        "Not yet published.",
		PostType:    "discovery",
		Visibility:  "public",
		Labels:      []string{"test"},
		ScheduledAt: &futureTime,
	})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if post2.Status != "scheduled" {
		t.Fatalf("expected status=scheduled, got %s", post2.Status)
	}

	// Move scheduled_at to the past to simulate time passing
	_, err = db.Exec("UPDATE posts SET scheduled_at = NOW() - INTERVAL '1 minute' WHERE id = $1", post2.ID)
	if err != nil {
		t.Fatalf("update scheduled_at: %v", err)
	}

	// Run publisher
	n, err := postRepo.PublishScheduled()
	if err != nil {
		t.Fatalf("publish scheduled: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 published, got %d", n)
	}

	// Verify post is now published
	published, err := postRepo.GetByID(post2.ID)
	if err != nil {
		t.Fatalf("get post: %v", err)
	}
	if published.Status != "published" {
		t.Errorf("expected status=published after worker, got %s", published.Status)
	}
}
