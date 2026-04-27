#!/usr/bin/env python3
"""
Tests for llm_compare_report.py scoring logic.
Run: python3 -m pytest test_report.py -v
  or: python3 test_report.py
"""

import sys
import os
import unittest

sys.path.insert(0, os.path.dirname(__file__))

from llm_compare_report import (
    score_rule_following,
    score_image_quality_deterministic,
    strip_utility_labels,
)

MINIMAL_EXPECTATIONS = {
    "_valid_post_types": ["article", "place", "discovery", "event", "video"],
    "_valid_display_hints": [
        "card", "article", "comparison", "digest", "brief", "science",
        "matchup", "concert", "restaurant", "destination", "fitness",
        "entertainment", "place",
    ],
    "hints": {},
}


class TestStripUtilityLabels(unittest.TestCase):
    """#246 — utility labels should be stripped before evaluation."""

    def test_strips_llm_compare_label(self):
        labels = ["tech", "news", "llm-compare"]
        self.assertEqual(strip_utility_labels(labels), ["tech", "news"])

    def test_strips_test_model_label(self):
        labels = ["sports", "nba", "test-sonnet46"]
        self.assertEqual(strip_utility_labels(labels), ["sports", "nba"])

    def test_strips_any_test_prefix_label(self):
        labels = ["food", "test-deepseek", "test-gemini"]
        self.assertEqual(strip_utility_labels(labels), ["food"])

    def test_preserves_non_utility_labels(self):
        labels = ["tech", "news", "science", "ai"]
        self.assertEqual(strip_utility_labels(labels), ["tech", "news", "science", "ai"])

    def test_empty_labels(self):
        self.assertEqual(strip_utility_labels([]), [])


class TestRuleFollowingUtilityLabelStripping(unittest.TestCase):
    """#246 — score_rule_following must strip utility labels before counting."""

    def _make_post(self, labels):
        return {
            "title": "Test Post Title",
            "body": "This is a body that is definitely longer than fifty characters total.",
            "image_url": "https://upload.wikimedia.org/img.jpg",
            "post_type": "article",
            "display_hint": "article",
            "labels": labels,
        }

    def test_nine_labels_with_two_utility_scores_full_label_check(self):
        # 9 labels total but 2 are utility → 7 real → in range [3,8] → should pass
        post = self._make_post([
            "tech", "news", "science", "health", "ai",
            "research", "coding",
            "llm-compare", "test-sonnet46",
        ])
        score = score_rule_following(post, MINIMAL_EXPECTATIONS)
        self.assertEqual(score, 10.0)

    def test_two_real_labels_with_two_utility_fails_label_check(self):
        # 4 labels total but 2 utility → 2 real → below minimum of 3 → should fail
        post = self._make_post(["tech", "news", "llm-compare", "test-sonnet46"])
        score = score_rule_following(post, MINIMAL_EXPECTATIONS)
        self.assertLess(score, 10.0)

    def test_eight_real_labels_no_utility_passes(self):
        post = self._make_post(["a", "b", "c", "d", "e", "f", "g", "h"])
        score = score_rule_following(post, MINIMAL_EXPECTATIONS)
        self.assertEqual(score, 10.0)


class TestScoreImageQualityDeterministic(unittest.TestCase):
    """#244 #245 — deterministic image quality scoring."""

    def _post(self, image_url):
        return {
            "title": "Post",
            "body": "body",
            "image_url": image_url,
            "labels": ["tech"],
        }

    def test_wikimedia_url_scores_high(self):
        posts = [self._post("https://upload.wikimedia.org/wikipedia/commons/img.jpg")]
        score = score_image_quality_deterministic(posts)
        self.assertGreaterEqual(score, 8.0)

    def test_unsplash_url_scores_high(self):
        posts = [self._post("https://images.unsplash.com/photo-abc123?w=800")]
        score = score_image_quality_deterministic(posts)
        self.assertGreaterEqual(score, 8.0)

    def test_pollinations_url_penalized(self):
        posts = [self._post("https://image.pollinations.ai/prompt/something?width=1024")]
        score = score_image_quality_deterministic(posts)
        self.assertLess(score, 5.0)

    def test_pollinations_gen_subdomain_penalized(self):
        posts = [self._post("https://gen.pollinations.ai/prompt/foo")]
        score = score_image_quality_deterministic(posts)
        self.assertLess(score, 5.0)

    def test_dalle_url_penalized(self):
        posts = [self._post("https://oaidalleapiprodscus.blob.core.windows.net/private/foo.png")]
        score = score_image_quality_deterministic(posts)
        self.assertLess(score, 5.0)

    def test_missing_image_url_scores_zero(self):
        posts = [{"title": "Post", "body": "body", "labels": ["tech"]}]
        score = score_image_quality_deterministic(posts)
        self.assertEqual(score, 0.0)

    def test_duplicate_image_urls_penalized(self):
        url = "https://upload.wikimedia.org/wikipedia/commons/img.jpg"
        posts = [self._post(url), self._post(url), self._post(url)]
        score = score_image_quality_deterministic(posts)
        self.assertLess(score, 8.0)

    def test_all_unique_wikimedia_scores_full(self):
        posts = [
            self._post("https://upload.wikimedia.org/wikipedia/commons/a.jpg"),
            self._post("https://upload.wikimedia.org/wikipedia/commons/b.jpg"),
            self._post("https://upload.wikimedia.org/wikipedia/commons/c.jpg"),
        ]
        score = score_image_quality_deterministic(posts)
        self.assertGreaterEqual(score, 8.0)

    def test_mixed_real_and_ai_partial_score(self):
        posts = [
            self._post("https://upload.wikimedia.org/wikipedia/commons/img.jpg"),
            self._post("https://image.pollinations.ai/prompt/foo"),
        ]
        score = score_image_quality_deterministic(posts)
        # Should be between low and high
        self.assertGreater(score, 0.0)
        self.assertLess(score, 8.0)

    def test_empty_posts_returns_zero(self):
        self.assertEqual(score_image_quality_deterministic([]), 0.0)


if __name__ == "__main__":
    unittest.main(verbosity=2)
