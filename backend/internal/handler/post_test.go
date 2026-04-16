package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
