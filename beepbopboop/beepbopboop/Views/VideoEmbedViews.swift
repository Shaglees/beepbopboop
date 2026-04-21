import SwiftUI
import WebKit

// MARK: - WebView

/// Loads the provider embed in an HTML iframe. Navigating the embed URL directly in WKWebView
/// often shows “Watch on YouTube” / broken playback; iframe matches normal web embed behavior.
struct VideoEmbedWebView: UIViewRepresentable {
    let embedURL: URL
    let provider: String

    func makeCoordinator() -> Coordinator {
        Coordinator()
    }

    /// Avoids reloading the iframe on every SwiftUI pass (fixes flicker / broken loads).
    final class Coordinator {
        var lastLoadedSrc: String?
    }

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
        let src = preparedEmbedURLString()
        if context.coordinator.lastLoadedSrc == src { return }
        context.coordinator.lastLoadedSrc = src

        let safe = Self.htmlEscapeAttribute(src)
        let html = """
        <!DOCTYPE html>
        <html><head>
        <meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1">
        <style>
          html, body { margin:0; padding:0; background:#000; height:100%; }
          .wrap { position:absolute; left:0; top:0; right:0; bottom:0; }
          iframe { position:absolute; left:0; top:0; width:100%; height:100%; border:0; }
        </style>
        </head><body>
        <div class="wrap">
        <iframe src="\(safe)"
          referrerpolicy="strict-origin-when-cross-origin"
          allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share; fullscreen"
          allowfullscreen></iframe>
        </div>
        </body></html>
        """
        let base = embedURL.host.flatMap { URL(string: "https://\($0)/") } ?? embedURL
        uiView.loadHTMLString(html, baseURL: base)
    }

    /// Tweaks embed URLs for inline playback (YouTube) and Vimeo’s player.
    private func preparedEmbedURLString() -> String {
        guard var components = URLComponents(url: embedURL, resolvingAgainstBaseURL: false) else {
            return embedURL.absoluteString
        }
        var items = components.queryItems ?? []

        switch provider {
        case "youtube":
            if !items.contains(where: { $0.name == "playsinline" }) {
                items.append(URLQueryItem(name: "playsinline", value: "1"))
            }
            if !items.contains(where: { $0.name == "rel" }) {
                items.append(URLQueryItem(name: "rel", value: "0"))
            }
        default:
            break
        }

        components.queryItems = items
        return components.url?.absoluteString ?? embedURL.absoluteString
    }

    private static func htmlEscapeAttribute(_ s: String) -> String {
        s.replacingOccurrences(of: "&", with: "&amp;")
            .replacingOccurrences(of: "\"", with: "&quot;")
    }
}

// MARK: - Feed card

struct VideoEmbedCard: View {
    let post: Post
    let embed: VideoEmbedData

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
                VideoEmbedWebView(embedURL: embedURL, provider: embed.provider)
                    .frame(maxWidth: .infinity)
                    .aspectRatio(16 / 9, contentMode: .fit)
                    .clipShape(RoundedRectangle(cornerRadius: 10))
            }

            CardFooter(post: post)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
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
                    VideoEmbedWebView(embedURL: url, provider: data.provider)
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
