import SwiftUI
import PhotosUI
import UIKit

// MARK: - ImagePicker

struct ImagePicker: UIViewControllerRepresentable {
    @Binding var imageData: Data?
    @Environment(\.dismiss) private var dismiss

    func makeCoordinator() -> Coordinator {
        Coordinator(self)
    }

    func makeUIViewController(context: Context) -> PHPickerViewController {
        var config = PHPickerConfiguration()
        config.filter = .images
        config.selectionLimit = 1
        let picker = PHPickerViewController(configuration: config)
        picker.delegate = context.coordinator
        return picker
    }

    func updateUIViewController(_ uiViewController: PHPickerViewController, context: Context) {}

    class Coordinator: NSObject, PHPickerViewControllerDelegate {
        let parent: ImagePicker

        init(_ parent: ImagePicker) {
            self.parent = parent
        }

        func picker(_ picker: PHPickerViewController, didFinishPicking results: [PHPickerResult]) {
            parent.dismiss()
            guard let provider = results.first?.itemProvider,
                  provider.canLoadObject(ofClass: UIImage.self) else { return }
            provider.loadObject(ofClass: UIImage.self) { object, _ in
                if let image = object as? UIImage,
                   let jpeg = image.jpegData(compressionQuality: 0.85) {
                    DispatchQueue.main.async {
                        self.parent.imageData = jpeg
                    }
                }
            }
        }
    }
}

// MARK: - ProfileView

struct ProfileView: View {
    @EnvironmentObject var apiService: APIService
    @State private var profile: UserProfile?
    @State private var isLoading = true

    // Photo state
    @State private var headshotData: Data?
    @State private var fullBodyData: Data?
    @State private var showHeadshotPicker = false
    @State private var showFullBodyPicker = false
    @State private var photoUploadError: String?
    @State private var isUploadingHeadshot = false
    @State private var isUploadingFullBody = false

    var body: some View {
        NavigationStack {
            Group {
                if isLoading {
                    ProgressView()
                } else if let profile {
                    ScrollView {
                        VStack(alignment: .leading, spacing: 24) {
                            // Identity section
                            VStack(alignment: .leading, spacing: 8) {
                                Text("PROFILE")
                                    .font(.system(size: 11, weight: .medium, design: .monospaced))
                                    .foregroundStyle(.secondary)
                                HStack(spacing: 12) {
                                    Circle()
                                        .fill(Color(.systemGray4))
                                        .frame(width: 48, height: 48)
                                        .overlay(
                                            Text(String(profile.identity.displayName.prefix(1)).uppercased())
                                                .font(.system(size: 20, weight: .bold, design: .serif))
                                        )
                                    VStack(alignment: .leading, spacing: 2) {
                                        Text(profile.identity.displayName)
                                            .font(.system(size: 17, weight: .semibold))
                                        Text("\(profile.identity.homeLocation) · \(profile.identity.timezone)")
                                            .font(.system(size: 13, design: .monospaced))
                                            .foregroundStyle(.secondary)
                                    }
                                }
                            }
                            .padding(.horizontal)

                            // My Photos section
                            VStack(alignment: .leading, spacing: 12) {
                                Text("MY PHOTOS")
                                    .font(.system(size: 11, weight: .medium, design: .monospaced))
                                    .foregroundStyle(.secondary)

                                HStack(spacing: 16) {
                                    // Headshot slot
                                    photoSlot(
                                        label: "Headshot",
                                        subtitle: "360 × 360",
                                        imageData: headshotData,
                                        isUploading: isUploadingHeadshot,
                                        onUpload: { showHeadshotPicker = true },
                                        onRemove: { Task { await removePhoto(type: "headshot") } }
                                    )

                                    // Full body slot
                                    photoSlot(
                                        label: "Full Body",
                                        subtitle: "360 × 720",
                                        imageData: fullBodyData,
                                        isUploading: isUploadingFullBody,
                                        onUpload: { showFullBodyPicker = true },
                                        onRemove: { Task { await removePhoto(type: "fullbody") } }
                                    )
                                }

                                if let err = photoUploadError {
                                    Text(err)
                                        .font(.caption)
                                        .foregroundStyle(.red)
                                }

                                Text("Photos are used for AI outfit try-on previews. Only you can manage them.")
                                    .font(.system(size: 11))
                                    .foregroundStyle(.secondary)
                            }
                            .padding(.horizontal)

                            // Interests section
                            if !profile.interests.isEmpty {
                                VStack(alignment: .leading, spacing: 8) {
                                    Text("INTERESTS")
                                        .font(.system(size: 11, weight: .medium, design: .monospaced))
                                        .foregroundStyle(.secondary)
                                    FlowLayout(spacing: 8) {
                                        ForEach(profile.interests) { interest in
                                            HStack(spacing: 4) {
                                                Text(interest.topic)
                                                    .font(.system(size: 13))
                                                if interest.source == "inferred" {
                                                    Image(systemName: "sparkles")
                                                        .font(.system(size: 9))
                                                        .foregroundStyle(.secondary)
                                                }
                                                if interest.pausedUntil != nil {
                                                    Image(systemName: "pause.circle")
                                                        .font(.system(size: 9))
                                                        .foregroundStyle(.orange)
                                                }
                                            }
                                            .padding(.horizontal, 12)
                                            .padding(.vertical, 6)
                                            .background(Color(.systemGray6))
                                            .clipShape(Capsule())
                                        }
                                    }
                                }
                                .padding(.horizontal)
                            }

                            // Lifestyle section
                            if !profile.lifestyle.isEmpty {
                                VStack(alignment: .leading, spacing: 8) {
                                    Text("LIFESTYLE")
                                        .font(.system(size: 11, weight: .medium, design: .monospaced))
                                        .foregroundStyle(.secondary)
                                    FlowLayout(spacing: 8) {
                                        ForEach(profile.lifestyle, id: \.value) { tag in
                                            Text(tag.value.replacingOccurrences(of: "_", with: " ").capitalized)
                                                .font(.system(size: 13))
                                                .padding(.horizontal, 12)
                                                .padding(.vertical, 6)
                                                .background(Color(.systemGray6))
                                                .clipShape(Capsule())
                                        }
                                    }
                                }
                                .padding(.horizontal)
                            }

                            // Content prefs section
                            if !profile.contentPrefs.isEmpty {
                                VStack(alignment: .leading, spacing: 8) {
                                    Text("CONTENT PREFERENCES")
                                        .font(.system(size: 11, weight: .medium, design: .monospaced))
                                        .foregroundStyle(.secondary)
                                    ForEach(Array(profile.contentPrefs.enumerated()), id: \.offset) { _, pref in
                                        HStack {
                                            Text(pref.category ?? "Global")
                                                .font(.system(size: 14, weight: .medium))
                                            Spacer()
                                            Text("\(pref.depth) · \(pref.tone)")
                                                .font(.system(size: 13, design: .monospaced))
                                                .foregroundStyle(.secondary)
                                            if let max = pref.maxPerDay {
                                                Text("≤\(max)/day")
                                                    .font(.system(size: 13, design: .monospaced))
                                                    .foregroundStyle(.secondary)
                                            }
                                        }
                                    }
                                }
                                .padding(.horizontal)
                            }
                        }
                        .padding(.vertical)
                    }
                } else {
                    Text("Failed to load profile")
                        .foregroundStyle(.secondary)
                }
            }
            .navigationTitle("Profile")
            .task { await loadProfile() }
            .task { await loadPhotos() }
            .sheet(isPresented: $showHeadshotPicker) {
                ImagePicker(imageData: $headshotData)
                    .onChange(of: headshotData) { _, newData in
                        if let data = newData {
                            Task { await uploadPhoto(type: "headshot", data: data) }
                        }
                    }
            }
            .sheet(isPresented: $showFullBodyPicker) {
                ImagePicker(imageData: $fullBodyData)
                    .onChange(of: fullBodyData) { _, newData in
                        if let data = newData {
                            Task { await uploadPhoto(type: "fullbody", data: data) }
                        }
                    }
            }
        }
    }

    @ViewBuilder
    private func photoSlot(
        label: String,
        subtitle: String,
        imageData: Data?,
        isUploading: Bool,
        onUpload: @escaping () -> Void,
        onRemove: @escaping () -> Void
    ) -> some View {
        VStack(spacing: 8) {
            // Thumbnail or placeholder
            ZStack {
                RoundedRectangle(cornerRadius: 8)
                    .fill(Color(.systemGray5))
                    .frame(width: 100, height: 120)

                if isUploading {
                    ProgressView()
                } else if let data = imageData, let uiImage = UIImage(data: data) {
                    Image(uiImage: uiImage)
                        .resizable()
                        .scaledToFill()
                        .frame(width: 100, height: 120)
                        .clipShape(RoundedRectangle(cornerRadius: 8))
                } else {
                    VStack(spacing: 4) {
                        Image(systemName: "person.crop.rectangle")
                            .font(.system(size: 24))
                            .foregroundStyle(.secondary)
                        Text(subtitle)
                            .font(.system(size: 9, design: .monospaced))
                            .foregroundStyle(.secondary)
                    }
                }
            }

            Text(label)
                .font(.system(size: 12, weight: .medium))

            HStack(spacing: 6) {
                Button(action: onUpload) {
                    Text("Upload")
                        .font(.system(size: 11))
                        .padding(.horizontal, 10)
                        .padding(.vertical, 4)
                        .background(Color.accentColor)
                        .foregroundStyle(.white)
                        .clipShape(Capsule())
                }

                if imageData != nil {
                    Button(action: onRemove) {
                        Text("Remove")
                            .font(.system(size: 11))
                            .padding(.horizontal, 10)
                            .padding(.vertical, 4)
                            .background(Color(.systemGray4))
                            .foregroundStyle(.primary)
                            .clipShape(Capsule())
                    }
                }
            }
        }
    }

    private func loadProfile() async {
        isLoading = true
        defer { isLoading = false }
        profile = try? await apiService.getProfile()
    }

    private func loadPhotos() async {
        async let headshot = try? await apiService.getPhoto(type: "headshot")
        async let fullBody = try? await apiService.getPhoto(type: "fullbody")
        headshotData = await headshot
        fullBodyData = await fullBody
    }

    private func uploadPhoto(type: String, data: Data) async {
        photoUploadError = nil
        if type == "headshot" { isUploadingHeadshot = true }
        else { isUploadingFullBody = true }
        defer {
            if type == "headshot" { isUploadingHeadshot = false }
            else { isUploadingFullBody = false }
        }
        do {
            try await apiService.uploadPhoto(type: type, imageData: data)
        } catch {
            photoUploadError = "Upload failed: \(error.localizedDescription)"
        }
    }

    private func removePhoto(type: String) async {
        photoUploadError = nil
        do {
            try await apiService.deletePhoto(type: type)
            if type == "headshot" { headshotData = nil }
            else { fullBodyData = nil }
        } catch {
            photoUploadError = "Remove failed: \(error.localizedDescription)"
        }
    }
}
