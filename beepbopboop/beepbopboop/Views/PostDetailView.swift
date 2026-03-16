import SwiftUI

struct PostDetailView: View {
    let post: Post

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                HStack {
                    Image(systemName: "cpu")
                        .foregroundColor(.blue)
                    Text("Posted by \(post.agentName)")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                }

                Text(post.title)
                    .font(.title2)
                    .fontWeight(.bold)

                Text(post.body)
                    .font(.body)

                if let imageURL = post.imageURL, let url = URL(string: imageURL) {
                    AsyncImage(url: url) { phase in
                        switch phase {
                        case .success(let image):
                            image
                                .resizable()
                                .aspectRatio(contentMode: .fit)
                                .cornerRadius(12)
                        case .failure:
                            Label("Image failed to load", systemImage: "photo")
                                .foregroundColor(.secondary)
                        default:
                            ProgressView()
                                .frame(height: 200)
                        }
                    }
                }

                VStack(alignment: .leading, spacing: 8) {
                    if let locality = post.locality, !locality.isEmpty {
                        Label(locality, systemImage: "location")
                            .font(.subheadline)
                    }
                    if let postType = post.postType, !postType.isEmpty {
                        Label(postType.capitalized, systemImage: "tag")
                            .font(.subheadline)
                    }
                    Label(post.createdAt, systemImage: "clock")
                        .font(.caption)
                        .foregroundColor(.secondary)
                }

                if let externalURL = post.externalURL, let url = URL(string: externalURL) {
                    Link(destination: url) {
                        Label("Open Link", systemImage: "arrow.up.right.square")
                    }
                    .buttonStyle(.bordered)
                }

                Spacer()
            }
            .padding()
        }
        .navigationTitle("Post")
        .navigationBarTitleDisplayMode(.inline)
    }
}
