# Wave 4 Part B: Fashion Try-On

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let users upload headshot + bodyshot photos, then generate AI try-on outfit previews using those photos.

**Spec:** `docs/superpowers/specs/2026-04-25-wave4-new-features-design.md` (Sub-system B)

---

### Task 7: Database Schema — User Photo Columns

**Files:**
- Modify: `backend/internal/database/database.go:~397` (before `return db, nil`)

- [ ] **Step 1: Add migration statements**

Add before the final `return db, nil` in the `Open` function:

```go
	// Wave 4: user photo storage
	db.ExecContext(ctx, `ALTER TABLE users ADD COLUMN IF NOT EXISTS headshot_data BYTEA`)
	db.ExecContext(ctx, `ALTER TABLE users ADD COLUMN IF NOT EXISTS headshot_type TEXT`)
	db.ExecContext(ctx, `ALTER TABLE users ADD COLUMN IF NOT EXISTS bodyshot_data BYTEA`)
	db.ExecContext(ctx, `ALTER TABLE users ADD COLUMN IF NOT EXISTS bodyshot_type TEXT`)
```

- [ ] **Step 2: Verify migration runs**

Run: `cd backend && go build ./cmd/server/`
Expected: Compiles.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/database/database.go
git commit -m "feat(wave4): add headshot/bodyshot columns to users table"
```

---

### Task 8: UserPhoto Repository

**Files:**
- Create: `backend/internal/repository/user_photo_repo.go`
- Create: `backend/internal/repository/user_photo_repo_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/repository/user_photo_repo_test.go`:

```go
package repository

import (
	"bytes"
	"testing"
)

func TestUserPhotoRepo_SaveAndGetHeadshot(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserPhotoRepo(db)
	userID := createTestUser(t, db)

	fakeJPEG := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10} // minimal JPEG header
	err := repo.SaveHeadshot(userID, fakeJPEG, "image/jpeg")
	if err != nil {
		t.Fatalf("SaveHeadshot: %v", err)
	}

	data, contentType, err := repo.GetHeadshot(userID)
	if err != nil {
		t.Fatalf("GetHeadshot: %v", err)
	}
	if contentType != "image/jpeg" {
		t.Errorf("expected image/jpeg, got %s", contentType)
	}
	if !bytes.Equal(data, fakeJPEG) {
		t.Error("headshot data mismatch")
	}
}

func TestUserPhotoRepo_SaveAndGetBodyshot(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserPhotoRepo(db)
	userID := createTestUser(t, db)

	fakeJPEG := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x20}
	err := repo.SaveBodyshot(userID, fakeJPEG, "image/jpeg")
	if err != nil {
		t.Fatalf("SaveBodyshot: %v", err)
	}

	data, contentType, err := repo.GetBodyshot(userID)
	if err != nil {
		t.Fatalf("GetBodyshot: %v", err)
	}
	if contentType != "image/jpeg" {
		t.Errorf("expected image/jpeg, got %s", contentType)
	}
	if !bytes.Equal(data, fakeJPEG) {
		t.Error("bodyshot data mismatch")
	}
}

func TestUserPhotoRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserPhotoRepo(db)
	userID := createTestUser(t, db)

	repo.SaveHeadshot(userID, []byte{0xFF}, "image/jpeg")

	err := repo.DeletePhoto(userID, "headshot")
	if err != nil {
		t.Fatalf("DeletePhoto: %v", err)
	}

	data, _, err := repo.GetHeadshot(userID)
	if err != nil {
		t.Fatalf("GetHeadshot after delete: %v", err)
	}
	if data != nil {
		t.Error("expected nil data after delete")
	}
}

func TestUserPhotoRepo_GetEmpty(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserPhotoRepo(db)
	userID := createTestUser(t, db)

	data, _, err := repo.GetHeadshot(userID)
	if err != nil {
		t.Fatalf("GetHeadshot: %v", err)
	}
	if data != nil {
		t.Error("expected nil data for user with no photo")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./internal/repository/ -run TestUserPhotoRepo -v`
Expected: FAIL — `NewUserPhotoRepo` not defined.

- [ ] **Step 3: Write the repository**

Create `backend/internal/repository/user_photo_repo.go`:

```go
package repository

import (
	"database/sql"
	"fmt"
)

type UserPhotoRepo struct {
	db *sql.DB
}

func NewUserPhotoRepo(db *sql.DB) *UserPhotoRepo {
	return &UserPhotoRepo{db: db}
}

func (r *UserPhotoRepo) SaveHeadshot(userID string, data []byte, contentType string) error {
	_, err := r.db.Exec(
		`UPDATE users SET headshot_data = $1, headshot_type = $2 WHERE id = $3`,
		data, contentType, userID,
	)
	return err
}

func (r *UserPhotoRepo) SaveBodyshot(userID string, data []byte, contentType string) error {
	_, err := r.db.Exec(
		`UPDATE users SET bodyshot_data = $1, bodyshot_type = $2 WHERE id = $3`,
		data, contentType, userID,
	)
	return err
}

func (r *UserPhotoRepo) GetHeadshot(userID string) ([]byte, string, error) {
	return r.getPhoto(userID, "headshot")
}

func (r *UserPhotoRepo) GetBodyshot(userID string) ([]byte, string, error) {
	return r.getPhoto(userID, "bodyshot")
}

func (r *UserPhotoRepo) getPhoto(userID, photoType string) ([]byte, string, error) {
	var data []byte
	var contentType sql.NullString

	col := photoType + "_data"
	typeCol := photoType + "_type"

	err := r.db.QueryRow(
		fmt.Sprintf(`SELECT %s, %s FROM users WHERE id = $1`, col, typeCol),
		userID,
	).Scan(&data, &contentType)

	if err == sql.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	return data, contentType.String, nil
}

func (r *UserPhotoRepo) DeletePhoto(userID, photoType string) error {
	col := photoType + "_data"
	typeCol := photoType + "_type"

	_, err := r.db.Exec(
		fmt.Sprintf(`UPDATE users SET %s = NULL, %s = NULL WHERE id = $1`, col, typeCol),
		userID,
	)
	return err
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/repository/ -run TestUserPhotoRepo -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/repository/user_photo_repo.go backend/internal/repository/user_photo_repo_test.go
git commit -m "feat(wave4): add UserPhotoRepo for headshot/bodyshot storage"
```

---

### Task 9: Photo Handler + Routes

**Files:**
- Create: `backend/internal/handler/photo.go`
- Create: `backend/internal/handler/photo_test.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/handler/photo_test.go`:

```go
package handler

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPhotoHandler_UploadHeadshot(t *testing.T) {
	srv := setupTestServer(t)

	// Build multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("photo", "headshot.jpg")
	// Minimal valid JPEG (just header bytes for test)
	part.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46})
	writer.Close()

	req := httptest.NewRequest("PUT", "/user/photos/headshot", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = injectFirebaseAuth(req, srv.firebaseUID)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPhotoHandler_GetHeadshot(t *testing.T) {
	srv := setupTestServer(t)

	// Upload first
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("photo", "headshot.jpg")
	part.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0})
	writer.Close()

	req := httptest.NewRequest("PUT", "/user/photos/headshot", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = injectFirebaseAuth(req, srv.firebaseUID)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	// Get it back
	req = httptest.NewRequest("GET", "/user/photos/headshot", nil)
	req = injectFirebaseAuth(req, srv.firebaseUID)
	w = httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/jpeg" {
		t.Errorf("expected image/jpeg, got %s", ct)
	}
}

func TestPhotoHandler_GetEmpty(t *testing.T) {
	srv := setupTestServer(t)

	req := httptest.NewRequest("GET", "/user/photos/headshot", nil)
	req = injectFirebaseAuth(req, srv.firebaseUID)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for no photo, got %d", w.Code)
	}
}

func TestPhotoHandler_AgentReadOnly(t *testing.T) {
	srv := setupTestServer(t)

	// Agent should NOT be able to upload
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("photo", "headshot.jpg")
	part.Write([]byte{0xFF, 0xD8})
	writer.Close()

	req := httptest.NewRequest("PUT", "/user/photos/headshot", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = injectAgentAuth(req, srv.agentID)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	// Should be 404 (route not registered for agent) or 405
	if w.Code == http.StatusOK {
		t.Fatal("agent should not be able to upload photos")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/handler/ -run TestPhotoHandler -v`
Expected: FAIL.

- [ ] **Step 3: Write the handler**

Create `backend/internal/handler/photo.go`:

```go
package handler

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

const maxPhotoSize = 1 << 20 // 1MB

type PhotoHandler struct {
	userRepo  *repository.UserRepo
	photoRepo *repository.UserPhotoRepo
}

func NewPhotoHandler(userRepo *repository.UserRepo, photoRepo *repository.UserPhotoRepo) *PhotoHandler {
	return &PhotoHandler{userRepo: userRepo, photoRepo: photoRepo}
}

func (h *PhotoHandler) UploadHeadshot(w http.ResponseWriter, r *http.Request) {
	h.upload(w, r, "headshot")
}

func (h *PhotoHandler) UploadBodyshot(w http.ResponseWriter, r *http.Request) {
	h.upload(w, r, "bodyshot")
}

func (h *PhotoHandler) upload(w http.ResponseWriter, r *http.Request, photoType string) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxPhotoSize)

	file, _, err := r.FormFile("photo")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing or too large photo field"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read photo"})
		return
	}

	// Store as-is for now. Server-side resize (Task for later or use image stdlib)
	// can be added here with golang.org/x/image/draw.
	contentType := "image/jpeg"

	if photoType == "headshot" {
		err = h.photoRepo.SaveHeadshot(user.ID, data, contentType)
	} else {
		err = h.photoRepo.SaveBodyshot(user.ID, data, contentType)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save photo"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *PhotoHandler) GetHeadshot(w http.ResponseWriter, r *http.Request) {
	h.getPhoto(w, r, "headshot")
}

func (h *PhotoHandler) GetBodyshot(w http.ResponseWriter, r *http.Request) {
	h.getPhoto(w, r, "bodyshot")
}

func (h *PhotoHandler) getPhoto(w http.ResponseWriter, r *http.Request, photoType string) {
	// Support both Firebase and Agent auth
	uid := middleware.FirebaseUIDFromContext(r.Context())
	var userID string
	if uid != "" {
		user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
			return
		}
		userID = user.ID
	} else {
		// Agent auth — resolve via agent's owner
		agentID := middleware.AgentIDFromContext(r.Context())
		if agentID == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		userID = middleware.UserIDFromContext(r.Context())
	}

	var data []byte
	var contentType string
	var err error
	if photoType == "headshot" {
		data, contentType, err = h.photoRepo.GetHeadshot(userID)
	} else {
		data, contentType, err = h.photoRepo.GetBodyshot(userID)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get photo"})
		return
	}
	if data == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no photo"})
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *PhotoHandler) DeletePhoto(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
		return
	}

	photoType := chi.URLParam(r, "type")
	if photoType != "headshot" && photoType != "bodyshot" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "type must be headshot or bodyshot"})
		return
	}

	if err := h.photoRepo.DeletePhoto(user.ID, photoType); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
```

- [ ] **Step 4: Wire routes in main.go**

Add to repos section:
```go
	photoRepo := repository.NewUserPhotoRepo(db)
```

Add to handlers section:
```go
	photoH := handler.NewPhotoHandler(userRepo, photoRepo)
```

Add to Firebase-auth routes:
```go
	r.Put("/user/photos/headshot", photoH.UploadHeadshot)
	r.Put("/user/photos/bodyshot", photoH.UploadBodyshot)
	r.Get("/user/photos/headshot", photoH.GetHeadshot)
	r.Get("/user/photos/bodyshot", photoH.GetBodyshot)
	r.Delete("/user/photos/{type}", photoH.DeletePhoto)
```

Add to agent-auth routes (read-only):
```go
	r.Get("/user/photos/headshot", photoH.GetHeadshot)
	r.Get("/user/photos/bodyshot", photoH.GetBodyshot)
```

- [ ] **Step 5: Run tests**

Run: `cd backend && go test ./internal/handler/ -run TestPhotoHandler -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/handler/photo.go backend/internal/handler/photo_test.go backend/cmd/server/main.go
git commit -m "feat(wave4): add photo upload/download/delete handler with routes"
```

---

### Task 10: iOS — Photo Upload UI + OutfitCard Try-On Label

**Files:**
- Modify: `beepbopboop/beepbopboop/Services/APIService.swift`
- Modify: `beepbopboop/beepbopboop/Views/ProfileView.swift`
- Modify: `beepbopboop/beepbopboop/Views/FeedItemView.swift` (OutfitCard area)

- [ ] **Step 1: Add photo methods to APIService**

Add to `APIService.swift` (before the closing `}`):

```swift
    // MARK: - Photos

    @MainActor
    func uploadPhoto(type photoType: String, imageData: Data) async throws {
        let token = authService.getToken()
        guard let url = URL(string: "\(baseURL)/user/photos/\(photoType)") else {
            throw APIError.invalidURL
        }

        let boundary = UUID().uuidString
        var request = URLRequest(url: url)
        request.httpMethod = "PUT"
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        request.setValue("multipart/form-data; boundary=\(boundary)", forHTTPHeaderField: "Content-Type")

        var body = Data()
        body.append("--\(boundary)\r\n".data(using: .utf8)!)
        body.append("Content-Disposition: form-data; name=\"photo\"; filename=\"\(photoType).jpg\"\r\n".data(using: .utf8)!)
        body.append("Content-Type: image/jpeg\r\n\r\n".data(using: .utf8)!)
        body.append(imageData)
        body.append("\r\n--\(boundary)--\r\n".data(using: .utf8)!)
        request.httpBody = body

        let (_, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }
        guard httpResponse.statusCode == 200 else {
            throw APIError.httpError(httpResponse.statusCode)
        }
    }

    @MainActor
    func getPhoto(type photoType: String) async throws -> Data? {
        let token = authService.getToken()
        guard let url = URL(string: "\(baseURL)/user/photos/\(photoType)") else {
            throw APIError.invalidURL
        }

        var request = URLRequest(url: url)
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")

        let (data, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }
        if httpResponse.statusCode == 404 {
            return nil
        }
        guard httpResponse.statusCode == 200 else {
            throw APIError.httpError(httpResponse.statusCode)
        }
        return data
    }

    @MainActor
    func deletePhoto(type photoType: String) async throws {
        let token = authService.getToken()
        guard let url = URL(string: "\(baseURL)/user/photos/\(photoType)") else {
            throw APIError.invalidURL
        }

        var request = URLRequest(url: url)
        request.httpMethod = "DELETE"
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")

        let (_, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }
        guard httpResponse.statusCode == 200 else {
            throw APIError.httpError(httpResponse.statusCode)
        }
    }
```

- [ ] **Step 2: Add "My Photos" section to ProfileView**

In `ProfileView.swift`, add a new section after the existing sections:

```swift
    // MARK: - My Photos Section

    Section("My Photos") {
        PhotoUploadRow(
            label: "Headshot",
            subtitle: "360×360 — used for AI outfit previews",
            photoType: "headshot",
            apiService: apiService
        )
        PhotoUploadRow(
            label: "Full Body",
            subtitle: "360×720 — used for AI outfit previews",
            photoType: "bodyshot",
            apiService: apiService
        )

        Text("Photos are used for AI outfit previews and stored on your account. Delete anytime.")
            .font(.caption2)
            .foregroundColor(.secondary)
    }
```

And define `PhotoUploadRow` as a separate struct in the same file or a new file:

```swift
struct PhotoUploadRow: View {
    let label: String
    let subtitle: String
    let photoType: String
    let apiService: APIService

    @State private var photoData: Data?
    @State private var showingPicker = false
    @State private var isLoading = false

    var body: some View {
        HStack {
            if let data = photoData, let uiImage = UIImage(data: data) {
                Image(uiImage: uiImage)
                    .resizable()
                    .aspectRatio(contentMode: .fill)
                    .frame(width: 50, height: 50)
                    .clipShape(RoundedRectangle(cornerRadius: 8))
            } else {
                RoundedRectangle(cornerRadius: 8)
                    .fill(Color.gray.opacity(0.2))
                    .frame(width: 50, height: 50)
                    .overlay(Image(systemName: "camera").foregroundColor(.gray))
            }

            VStack(alignment: .leading) {
                Text(label).font(.body)
                Text(subtitle).font(.caption2).foregroundColor(.secondary)
            }

            Spacer()

            if isLoading {
                ProgressView()
            } else if photoData != nil {
                Button("Remove", role: .destructive) {
                    Task {
                        isLoading = true
                        try? await apiService.deletePhoto(type: photoType)
                        photoData = nil
                        isLoading = false
                    }
                }
                .font(.caption)
            } else {
                Button("Upload") { showingPicker = true }
                    .font(.caption)
            }
        }
        .sheet(isPresented: $showingPicker) {
            ImagePicker(imageData: $photoData, onPick: { data in
                Task {
                    isLoading = true
                    try? await apiService.uploadPhoto(type: photoType, imageData: data)
                    isLoading = false
                }
            })
        }
        .task {
            photoData = try? await apiService.getPhoto(type: photoType)
        }
    }
}
```

**Note:** `ImagePicker` wraps `UIImagePickerController`. If one already exists in the codebase, reuse it. Otherwise create a minimal one:

```swift
struct ImagePicker: UIViewControllerRepresentable {
    @Binding var imageData: Data?
    var onPick: (Data) -> Void

    func makeUIViewController(context: Context) -> UIImagePickerController {
        let picker = UIImagePickerController()
        picker.delegate = context.coordinator
        picker.allowsEditing = true
        return picker
    }

    func updateUIViewController(_ uiViewController: UIImagePickerController, context: Context) {}

    func makeCoordinator() -> Coordinator { Coordinator(self) }

    class Coordinator: NSObject, UIImagePickerControllerDelegate, UINavigationControllerDelegate {
        let parent: ImagePicker
        init(_ parent: ImagePicker) { self.parent = parent }

        func imagePickerController(_ picker: UIImagePickerController,
                                   didFinishPickingMediaWithInfo info: [UIImagePickerController.InfoKey: Any]) {
            if let image = info[.editedImage] as? UIImage ?? info[.originalImage] as? UIImage,
               let data = image.jpegData(compressionQuality: 0.8) {
                parent.imageData = data
                parent.onPick(data)
            }
            picker.dismiss(animated: true)
        }

        func imagePickerControllerDidCancel(_ picker: UIImagePickerController) {
            picker.dismiss(animated: true)
        }
    }
}
```

- [ ] **Step 3: Add try-on label to OutfitCard**

In the OutfitCard section of `FeedItemView.swift` (~line 1197), find where the outfit image is rendered and add:

```swift
// Inside the outfit image ZStack, add:
if let extURL = post.externalURL,
   let data = extURL.data(using: .utf8),
   let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
   json["image_variant"] as? String == "tryon" {
    VStack {
        Spacer()
        HStack {
            Text("AI try-on preview")
                .font(.caption2)
                .foregroundColor(.white)
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(Color.black.opacity(0.5))
                .cornerRadius(8)
                .padding(8)
            Spacer()
        }
    }
}
```

- [ ] **Step 4: Build iOS**

Run:
```bash
xcodebuild -project beepbopboop/beepbopboop.xcodeproj -scheme beepbopboop -sdk iphonesimulator -destination 'platform=iOS Simulator,name=iPhone 17 Pro' -derivedDataPath /tmp/bbp-build clean build 2>&1 | tail -5
```
Expected: BUILD SUCCEEDED

- [ ] **Step 5: Commit**

```bash
git add beepbopboop/beepbopboop/Services/APIService.swift beepbopboop/beepbopboop/Views/ProfileView.swift beepbopboop/beepbopboop/Views/FeedItemView.swift
git commit -m "feat(wave4): add photo upload UI + outfit card try-on label"
```

---

### Task 11: `MODE_TRYON.md` Skill Mode

**Files:**
- Create: `.claude/skills/beepbopboop-fashion/MODE_TRYON.md`
- Modify: `.claude/skills/beepbopboop-fashion/SKILL.md` (add routing entry)

- [ ] **Step 1: Create MODE_TRYON.md**

Create `.claude/skills/beepbopboop-fashion/MODE_TRYON.md`:

```markdown
# Mode: Fashion Try-On

Generate an AI outfit preview using the user's uploaded bodyshot photo.

## Step 1: Check User Photo

```bash
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  "$BEEPBOPBOOP_API_URL/user/photos/bodyshot" \
  -H "Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN")
```

If `STATUS` is `404`: user has no bodyshot. **Fall back to standard outfit mode** — read `MODE_OUTFIT.md` instead and stop here.

If `STATUS` is `200`: download the photo:
```bash
curl -s "$BEEPBOPBOOP_API_URL/user/photos/bodyshot" \
  -H "Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN" \
  -o /tmp/bodyshot.jpg
```

## Step 2: Fetch Trends + Preferences

Reuse the existing fashion skill trend fetching (same as standard outfit mode):
- Read user fashion prefs from config: `BEEPBOPBOOP_FASHION_STYLES`, `BEEPBOPBOOP_FASHION_BUDGET`, `BEEPBOPBOOP_FASHION_BRANDS`
- Search current trends matching those preferences

## Step 3: Compose Outfit Description

Write a detailed text prompt describing the outfit:
- Style direction from preferences + trends
- Specific garments (top, bottom, shoes, accessories)
- Colors and materials
- Season-appropriate

## Step 4: Generate Try-On Image

Call OpenAI image generation with the bodyshot as reference:

```python
# Conceptual — the skill uses whatever image generation tool is available
# Input: /tmp/bodyshot.jpg + text prompt
# Output: AI-generated image of outfit on figure resembling user
```

**If image generation fails:** fall back to standard outfit post with text description only.

## Step 5: Upload Image

Use the `beepbopboop-images` skill/pipeline to upload the generated image to Imgur:

```bash
# Follow ../beepbopboop-images/SKILL.md for image hosting
```

**If upload fails:** use the direct image URL from the generation service (may be temporary).

## Step 6: Compose Post

```json
{
  "title": "Try-On: <outfit description>",
  "body": "<2-3 sentences about the outfit, why it works, where to wear it>",
  "post_type": "discovery",
  "display_hint": "outfit",
  "image_url": "<hosted image URL>",
  "external_url": "{\"image_variant\":\"tryon\",\"outfit_items\":[...],\"style\":\"...\"}",
  "labels": ["fashion", "try-on", "<style>"]
}
```

## Step 7: Lint + Publish

Follow `../_shared/PUBLISH_ENVELOPE.md`.
```

- [ ] **Step 2: Add routing entry to SKILL.md**

In `.claude/skills/beepbopboop-fashion/SKILL.md`, add to the mode routing table:

```markdown
| "try on" / "try-on" / "virtual fitting" | tryon | `MODE_TRYON.md` |
```

- [ ] **Step 3: Commit**

```bash
git add .claude/skills/beepbopboop-fashion/MODE_TRYON.md .claude/skills/beepbopboop-fashion/SKILL.md
git commit -m "feat(wave4): add fashion try-on skill mode"
```
