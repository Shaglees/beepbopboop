//
//  beepbopboopTests.swift
//  beepbopboopTests
//
//  Created by Shane Gleeson on 2026-03-15.
//

import Foundation
import Testing
@testable import beepbopboop

@MainActor
struct EventTrackerTests {

    // MARK: - Buffer & batching

    @Test func bufferAccumulatesEvents() async throws {
        var flushed: [[EventTracker.PendingEvent]] = []
        let tracker = EventTracker(flushThreshold: 10) { events in
            flushed.append(events)
        }

        tracker.fireEvent(postID: "a", type: "expand")
        tracker.fireEvent(postID: "b", type: "save")

        #expect(tracker.buffer.count == 2)
        #expect(flushed.isEmpty)
    }

    @Test func autoFlushesAtThreshold() async throws {
        var flushed: [[EventTracker.PendingEvent]] = []
        let tracker = EventTracker(flushThreshold: 3) { events in
            flushed.append(events)
        }

        tracker.fireEvent(postID: "a", type: "expand")
        tracker.fireEvent(postID: "b", type: "save")
        tracker.fireEvent(postID: "c", type: "view")

        // Give the auto-flush Task a chance to run
        try await Task.sleep(for: .milliseconds(50))

        #expect(flushed.count == 1)
        #expect(flushed[0].count == 3)
        #expect(tracker.buffer.isEmpty)
    }

    @Test func manualFlushDrainsBuffer() async throws {
        var flushed: [[EventTracker.PendingEvent]] = []
        let tracker = EventTracker(flushThreshold: 10) { events in
            flushed.append(events)
        }

        tracker.fireEvent(postID: "x", type: "save")
        await tracker.flush()

        #expect(flushed.count == 1)
        #expect(flushed[0].first?.eventType == "save")
        #expect(tracker.buffer.isEmpty)
    }

    @Test func flushOnEmptyBufferIsNoop() async throws {
        var callCount = 0
        let tracker = EventTracker(flushThreshold: 10) { _ in callCount += 1 }
        await tracker.flush()
        #expect(callCount == 0)
    }

    // MARK: - View event deduplication

    @Test func viewEventNotFiredTwiceForSamePost() async throws {
        var flushed: [[EventTracker.PendingEvent]] = []
        let tracker = EventTracker(flushThreshold: 10) { events in
            flushed.append(events)
        }

        // Simulate appear → disappear after 1.1s → reappear
        tracker.cardAppeared(postID: "post1")
        try await Task.sleep(for: .milliseconds(1100))
        tracker.cardDisappeared(postID: "post1")
        tracker.cardAppeared(postID: "post1") // reappear — should not schedule another view event

        try await Task.sleep(for: .milliseconds(1100))
        tracker.cardDisappeared(postID: "post1")

        let viewEvents = tracker.buffer.filter { $0.eventType == "view" }
        #expect(viewEvents.count == 1)
    }

    // MARK: - Dwell thresholds

    @Test func shortVisibilityFiresNoEvent() async throws {
        let tracker = EventTracker(flushThreshold: 10) { _ in }

        tracker.cardAppeared(postID: "fast")
        try await Task.sleep(for: .milliseconds(100))
        tracker.cardDisappeared(postID: "fast")

        #expect(tracker.buffer.isEmpty)
    }

    @Test func dwellEventRequiresThreeSeconds() async throws {
        let tracker = EventTracker(flushThreshold: 10) { _ in }

        // Visible for ~600ms — above 500ms min but below 3s dwell threshold
        tracker.cardAppeared(postID: "mid")
        try await Task.sleep(for: .milliseconds(600))
        tracker.cardDisappeared(postID: "mid")

        // Should have no dwell event (< 3s), but view timer was cancelled so no view event either
        let dwellEvents = tracker.buffer.filter { $0.eventType == "dwell" }
        #expect(dwellEvents.isEmpty)
    }

    // MARK: - Event encoding

    @Test func pendingEventEncodesCorrectly() throws {
        let event = EventTracker.PendingEvent(postID: "abc", eventType: "dwell", dwellMs: 4200)
        let data = try JSONEncoder().encode(event)
        let dict = try JSONSerialization.jsonObject(with: data) as? [String: Any]

        #expect(dict?["post_id"] as? String == "abc")
        #expect(dict?["event_type"] as? String == "dwell")
        #expect(dict?["dwell_ms"] as? Int == 4200)
    }
}

@MainActor
struct VideoEmbedPreviewCapTests {

    @Test func videoEmbedDataDecodesSupportsPreviewCap() throws {
        let json = """
        {
          "provider": "youtube",
          "video_id": "jNQXAC9IVRw",
          "watch_url": "https://www.youtube.com/watch?v=jNQXAC9IVRw",
          "embed_url": "https://www.youtube.com/embed/jNQXAC9IVRw",
          "supports_preview_cap": true
        }
        """
        let data = try #require(json.data(using: .utf8))
        let decoded = try JSONDecoder().decode(VideoEmbedData.self, from: data)
        #expect(decoded.previewCapEnabled == true)
    }

    @Test func previewCapHTMLInjectedForYouTube() throws {
        let url = try #require(URL(string: "https://www.youtube.com/embed/jNQXAC9IVRw"))
        let html = VideoEmbedHTMLBuilder.html(embedURL: url, provider: "youtube", previewCapSec: 60)
        #expect(html.contains("youtube.com/iframe_api"))
        #expect(html.contains("pauseVideo()"))
        #expect(html.contains(VideoEmbedHTMLBuilder.previewCapMessageName))
    }

    @Test func previewCapHTMLInjectedForVimeo() throws {
        let url = try #require(URL(string: "https://player.vimeo.com/video/1084537"))
        let html = VideoEmbedHTMLBuilder.html(embedURL: url, provider: "vimeo", previewCapSec: 60)
        #expect(html.contains("player.vimeo.com/api/player.js"))
        #expect(html.contains("timeupdate"))
        #expect(html.contains(VideoEmbedHTMLBuilder.previewCapMessageName))
    }

    @Test func playbackStateTransitionsOnCapReachedMessage() throws {
        let state = VideoEmbedPlaybackState()
        #expect(state.capReached == false)
        state.handleScriptMessage(name: VideoEmbedHTMLBuilder.previewCapMessageName, body: "capReached")
        #expect(state.capReached == true)
    }
}
