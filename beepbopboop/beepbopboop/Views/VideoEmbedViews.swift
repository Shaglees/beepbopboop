import SwiftUI
import WebKit

// MARK: - WebView

struct VideoEmbedWebView: UIViewRepresentable {
    let url: URL

    func makeUIView(context: Context) -> WKWebView {
        let config = WKWebViewConfiguration()
        config.allowsInlineMediaPlayback = true
        config.mediaTypesRequiringUserActionForPlayback = []
        let webView = WKWebView(frame: .zero, configuration: config)
        webView.scrollView.isScrollEnabled = false
        webView.isOpaque = false
        webView.backgroundColor = .clear
        return webView
    }

    func updateUIView(_ uiView: WKWebView, context: Context) {
        uiView.load(URLRequest(url: url))
    }
}

// MARK: - Feed card

struct VideoEmbedCard: View {
    let post: Post
    let embed: VideoEmbedData
    @State private var showPlayer = false

    init?(post: Post) {
        guard post.displayHintValue == .videoEmbed,
              let data = post.videoEmbedData else { return nil }
        self.post = post
        self.embed = data
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            CardHeader(post: post)

            Text(post.title)
                .font(.headline)
                .lineLimit(2)

            Text(post.body)
                .font(.subheadline)
                .foregroundColor(.secondary)
                .lineLimit(3)

            if let embedURL = URL(string: embed.embedUrl) {
                Button {
                    showPlayer = true
                } label: {
                    ZStack {
                        thumbnail
                        Image(systemName: "play.circle.fill")
                            .font(.system(size: 56))
                            .symbolRenderingMode(.palette)
                            .foregroundStyle(.white, .black.opacity(0.45))
                            .shadow(color: .black.opacity(0.35), radius: 6, y: 2)
                    }
                    .aspectRatio(16 / 9, contentMode: .fit)
                    .frame(maxWidth: .infinity)
                    .clipped()
                    .cornerRadius(10)
                }
                .buttonStyle(.plain)
                .sheet(isPresented: $showPlayer) {
                    NavigationStack {
                        VideoEmbedWebView(url: embedURL)
                            .ignoresSafeArea(edges: .bottom)
                            .navigationTitle(embed.channelTitle ?? "Video")
                            .navigationBarTitleDisplayMode(.inline)
                    }
                }
            }

            CardFooter(post: post)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
    }

    @ViewBuilder
    private var thumbnail: some View {
        if let thumb = embed.thumbnailUrl ?? post.imageURL,
           !thumb.isEmpty,
           let url = URL(string: thumb) {
            AsyncImage(url: url) { phase in
                switch phase {
                case .success(let image):
                    image
                        .resizable()
                        .aspectRatio(contentMode: .fill)
                case .failure:
                    Color.secondary.opacity(0.2)
                default:
                    Color.secondary.opacity(0.15)
                        .overlay { ProgressView() }
                }
            }
        } else {
            Color.secondary.opacity(0.2)
                .overlay {
                    Image(systemName: "play.rectangle")
                        .font(.largeTitle)
                        .foregroundStyle(.secondary)
                }
        }
    }
}

// MARK: - Detail

struct VideoEmbedDetailView: View {
    let post: Post
    @Environment(\.dismiss) private var dismiss

    private var embed: VideoEmbedData? { post.videoEmbedData }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                if let data = embed, let url = URL(string: data.embedUrl) {
                    VideoEmbedWebView(url: url)
                        .frame(height: 220)
                        .clipShape(RoundedRectangle(cornerRadius: 12))
                }

                VStack(alignment: .leading, spacing: 8) {
                    if let ch = embed?.channelTitle, !ch.isEmpty {
                        Text(ch)
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(.secondary)
                    }
                    Text(post.title)
                        .font(.title2.weight(.bold))
                    if !post.body.isEmpty {
                        Text(post.body)
                            .font(.body)
                            .lineSpacing(4)
                    }
                }

                Divider()
                PostDetailEngagementBar(post: post)
            }
            .padding(16)
        }
        .navigationTitle("")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button { dismiss() } label: {
                    Image(systemName: "xmark.circle.fill")
                        .foregroundStyle(.secondary)
                }
            }
        }
    }
}
