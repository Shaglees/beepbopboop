import SwiftUI
import WebKit

// MARK: - WebView

enum VideoEmbedHTMLBuilder {
    static let previewCapMessageName = "videoCap"
    static let defaultPreviewCapSec = 60

    static func preparedEmbedURLString(embedURL: URL, provider: String) -> String {
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

    static func html(embedURL: URL, provider: String, previewCapSec: Int?) -> String {
        let src = preparedEmbedURLString(embedURL: embedURL, provider: provider)
        let safeSrc = htmlEscapeAttribute(src)

        let capJS: String
        if let cap = previewCapSec, cap > 0 {
            switch provider {
            case "youtube":
                capJS = """
                <script src="https://www.youtube.com/iframe_api"></script>
                <script>
                  const PREVIEW_CAP_SEC = \(cap);
                  let capTriggered = false;
                  function notifyCapReached() {
                    if (capTriggered) return;
                    capTriggered = true;
                    if (window.webkit && window.webkit.messageHandlers && window.webkit.messageHandlers.\(previewCapMessageName)) {
                      window.webkit.messageHandlers.\(previewCapMessageName).postMessage('capReached');
                    }
                  }
                  function onYouTubeIframeAPIReady() {
                    const iframe = document.getElementById('video-embed-iframe');
                    if (!iframe) return;
                    const player = new YT.Player(iframe, {
                      events: {
                        onStateChange: function(event) {
                          if (event.data !== YT.PlayerState.PLAYING) return;
                          const timer = setInterval(function() {
                            const current = player.getCurrentTime ? player.getCurrentTime() : 0;
                            if (current >= PREVIEW_CAP_SEC) {
                              clearInterval(timer);
                              if (player.pauseVideo) { player.pauseVideo(); }
                              notifyCapReached();
                            }
                          }, 250);
                        }
                      }
                    });
                  }
                </script>
                """
            case "vimeo":
                capJS = """
                <script src="https://player.vimeo.com/api/player.js"></script>
                <script>
                  document.addEventListener('DOMContentLoaded', function() {
                    const iframe = document.getElementById('video-embed-iframe');
                    if (!iframe || typeof Vimeo === 'undefined') return;
                    const player = new Vimeo.Player(iframe);
                    let capTriggered = false;
                    player.on('timeupdate', function(data) {
                      if (capTriggered) return;
                      if ((data.seconds || 0) >= \(cap)) {
                        capTriggered = true;
                        player.pause();
                        if (window.webkit && window.webkit.messageHandlers && window.webkit.messageHandlers.\(previewCapMessageName)) {
                          window.webkit.messageHandlers.\(previewCapMessageName).postMessage('capReached');
                        }
                      }
                    });
                  });
                </script>
                """
            default:
                capJS = ""
            }
        } else {
            capJS = ""
        }

        return """
        <!DOCTYPE html>
        <html><head>
        <meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1">
        <style>
          html, body { margin:0; padding:0; background:#000; height:100%; }
          .wrap { position:absolute; left:0; top:0; right:0; bottom:0; }
          iframe { position:absolute; left:0; top:0; width:100%; height:100%; border:0; }
        </style>
        \(capJS)
        </head><body>
        <div class="wrap">
        <iframe id="video-embed-iframe" src="\(safeSrc)"
          referrerpolicy="strict-origin-when-cross-origin"
          allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share; fullscreen"
          allowfullscreen></iframe>
        </div>
        </body></html>
        """
    }

    private static func htmlEscapeAttribute(_ s: String) -> String {
        s.replacingOccurrences(of: "&", with: "&amp;")
            .replacingOccurrences(of: "\"", with: "&quot;")
    }
}

@MainActor
final class VideoEmbedPlaybackState: ObservableObject {
    @Published var capReached = false

    func handleScriptMessage(name: String, body: Any) {
        guard name == VideoEmbedHTMLBuilder.previewCapMessageName else { return }
        if let event = body as? String, event == "capReached" {
            capReached = true
        }
    }
}

/// Loads the provider embed in an HTML iframe. Navigating the embed URL directly in WKWebView
/// often shows “Watch on YouTube” / broken playback; iframe matches normal web embed behavior.
struct VideoEmbedWebView: UIViewRepresentable {
    let embedURL: URL
    let provider: String
    let previewCapSec: Int?
    let onCapReached: (() -> Void)?

    func makeCoordinator() -> Coordinator {
        Coordinator(onCapReached: onCapReached)
    }

    /// Avoids reloading the iframe on every SwiftUI pass (fixes flicker / broken loads).
    final class Coordinator: NSObject, WKScriptMessageHandler {
        var lastLoadedSrc: String?
        private let onCapReached: (() -> Void)?

        init(onCapReached: (() -> Void)?) {
            self.onCapReached = onCapReached
        }

        func userContentController(_ userContentController: WKUserContentController, didReceive message: WKScriptMessage) {
            guard message.name == VideoEmbedHTMLBuilder.previewCapMessageName else { return }
            if let event = message.body as? String, event == "capReached" {
                onCapReached?()
            }
        }
    }

    func makeUIView(context: Context) -> WKWebView {
        let config = WKWebViewConfiguration()
        config.allowsInlineMediaPlayback = true
        config.mediaTypesRequiringUserActionForPlayback = []
        config.userContentController.add(context.coordinator, name: VideoEmbedHTMLBuilder.previewCapMessageName)
        let webView = WKWebView(frame: .zero, configuration: config)
        webView.scrollView.isScrollEnabled = false
        webView.isOpaque = false
        webView.backgroundColor = .clear
        return webView
    }

    func updateUIView(_ uiView: WKWebView, context: Context) {
        let html = VideoEmbedHTMLBuilder.html(embedURL: embedURL, provider: provider, previewCapSec: previewCapSec)
        let loadKey = "\(provider)|\(embedURL.absoluteString)|\(previewCapSec.map(String.init) ?? "none")"
        if context.coordinator.lastLoadedSrc == loadKey { return }
        context.coordinator.lastLoadedSrc = loadKey
        let base = embedURL.host.flatMap { URL(string: "https://\($0)/") } ?? embedURL
        uiView.loadHTMLString(html, baseURL: base)
    }
}

// MARK: - Feed card

struct VideoEmbedCard: View {
    let post: Post
    let embed: VideoEmbedData
    @State private var capReached = false

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

            if capReached {
                previewCapCTA
            } else if let embedURL = URL(string: embed.embedUrl) {
                VideoEmbedWebView(
                    embedURL: embedURL,
                    provider: embed.provider,
                    previewCapSec: embed.previewCapEnabled ? VideoEmbedHTMLBuilder.defaultPreviewCapSec : nil,
                    onCapReached: { capReached = true }
                )
                .frame(maxWidth: .infinity)
                .aspectRatio(16 / 9, contentMode: .fit)
                .clipShape(RoundedRectangle(cornerRadius: 10))
            }

            CardFooter(post: post)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
    }

    @ViewBuilder
    private var previewCapCTA: some View {
        if let watch = embed.watchUrl, let url = URL(string: watch) {
            Link(destination: url) {
                previewCapLabel
            }
            .buttonStyle(.plain)
        } else {
            previewCapLabel
        }
    }

    private var previewCapLabel: some View {
        VStack(spacing: 10) {
            Image(systemName: "play.rectangle.fill")
                .font(.title2)
            Text("Preview complete")
                .font(.headline)
            Text("Watch full video")
                .font(.subheadline)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity)
        .frame(height: 220)
        .background(Color(.secondarySystemBackground))
        .clipShape(RoundedRectangle(cornerRadius: 10))
    }
}

// MARK: - Detail

struct VideoEmbedDetailView: View {
    let post: Post
    @Environment(\.dismiss) private var dismiss
    @State private var capReached = false

    private var embed: VideoEmbedData? { post.videoEmbedData }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                if let data = embed {
                    if capReached {
                        detailPreviewCapCTA(data: data)
                    } else if let url = URL(string: data.embedUrl) {
                        VideoEmbedWebView(
                            embedURL: url,
                            provider: data.provider,
                            previewCapSec: data.previewCapEnabled ? VideoEmbedHTMLBuilder.defaultPreviewCapSec : nil,
                            onCapReached: { capReached = true }
                        )
                        .frame(height: 220)
                        .clipShape(RoundedRectangle(cornerRadius: 12))
                    }
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

    @ViewBuilder
    private func detailPreviewCapCTA(data: VideoEmbedData) -> some View {
        if let watch = data.watchUrl, let url = URL(string: watch) {
            Link(destination: url) {
                VStack(spacing: 10) {
                    Image(systemName: "play.rectangle.fill")
                        .font(.title2)
                    Text("Preview complete")
                        .font(.headline)
                    Text("Watch full video")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }
                .frame(maxWidth: .infinity)
                .frame(height: 220)
                .background(Color(.secondarySystemBackground))
                .clipShape(RoundedRectangle(cornerRadius: 12))
            }
            .buttonStyle(.plain)
        }
    }
}
