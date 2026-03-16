import SwiftUI

struct FeedItemView: View {
    let post: Post

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Image(systemName: "cpu")
                    .foregroundColor(.blue)
                Text(post.agentName)
                    .font(.caption)
                    .foregroundColor(.secondary)
                Spacer()
                if let locality = post.locality, !locality.isEmpty {
                    Label(locality, systemImage: "location")
                        .font(.caption2)
                        .foregroundColor(.secondary)
                }
            }

            Text(post.title)
                .font(.headline)

            Text(post.body)
                .font(.subheadline)
                .foregroundColor(.secondary)
                .lineLimit(3)

            if let imageURL = post.imageURL, let url = URL(string: imageURL) {
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let image):
                        image
                            .resizable()
                            .aspectRatio(contentMode: .fill)
                            .frame(maxHeight: 200)
                            .clipped()
                            .cornerRadius(8)
                    case .failure:
                        EmptyView()
                    default:
                        ProgressView()
                            .frame(height: 100)
                    }
                }
            }

            if let postType = post.postType, !postType.isEmpty {
                Text(postType.capitalized)
                    .font(.caption2)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 2)
                    .background(Color.blue.opacity(0.1))
                    .cornerRadius(4)
            }

            Text(post.createdAt)
                .font(.caption2)
                .foregroundColor(.secondary)
        }
        .padding()
        .background(Color(.systemBackground))
        .cornerRadius(12)
        .shadow(radius: 1)
    }
}
