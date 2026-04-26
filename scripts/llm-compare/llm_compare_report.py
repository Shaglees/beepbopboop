#!/usr/bin/env python3
"""
LLM Model Comparison Report — Quality scoring engine.

Scores each model's posts across 7 dimensions and generates 3 report tables.
Pure stdlib Python (no pip install). Uses urllib for OpenRouter API calls.

Usage:
  python3 llm_compare_report.py <results_dir> <models_conf> \
    --expectations <expectations.json> \
    [--compare-model <openrouter_id>] \
    [--openrouter-key <key>]
"""

import argparse
import json
import math
import os
import sys
import urllib.request
import urllib.error
from collections import Counter, defaultdict

# ─── Constants ───

WEIGHTS = {
    "rule_following": 2,
    "render_expectation": 3,
    "content_diversity": 2,
    "non_repetition": 1,
    "profile_matching": 2,
    "sophistication": 1,
    "image_quality": 1,
}

DIMENSION_LABELS = {
    "rule_following": "Rules",
    "render_expectation": "Render",
    "content_diversity": "Divers",
    "non_repetition": "NoRep",
    "profile_matching": "Profile",
    "sophistication": "Sophis",
    "image_quality": "Images",
}


# ─── Data Loading ───

def parse_models_conf(path):
    """Parse models.conf into list of dicts."""
    models = []
    with open(path) as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            parts = line.split("|")
            if len(parts) >= 4:
                models.append({
                    "key": parts[0],
                    "openrouter_id": parts[1],
                    "display_name": parts[2],
                    "status": parts[3],
                })
    return models


def load_results(results_dir, model_key):
    """Load a model's results JSON. Returns list of posts or empty list."""
    path = os.path.join(results_dir, f"{model_key}.json")
    if not os.path.exists(path):
        return []
    try:
        with open(path) as f:
            data = json.load(f)
        return data if isinstance(data, list) else []
    except (json.JSONDecodeError, IOError):
        return []


def load_expectations(path):
    """Load expectations.json."""
    with open(path) as f:
        return json.load(f)


# ─── Deterministic Scoring ───

def score_rule_following(post, expectations):
    """Score a single post on rule following (0-10)."""
    checks = []
    valid_types = set(expectations.get("_valid_post_types", []))
    valid_hints = set(expectations.get("_valid_display_hints", []))

    # Has title
    checks.append(bool(post.get("title", "").strip()))

    # Has body > 50 chars
    body = post.get("body", "") or ""
    checks.append(len(body) > 50)

    # Has image
    checks.append(bool(post.get("image_url")))

    # Valid post_type
    pt = post.get("post_type", post.get("type", ""))
    checks.append(pt in valid_types)

    # Valid display_hint
    hint = post.get("display_hint", "")
    checks.append(hint in valid_hints)

    # Has labels (3-8)
    labels = post.get("labels", []) or []
    label_count = len(labels)
    checks.append(3 <= label_count <= 8)

    # Labels are lowercase-hyphenated
    if labels:
        valid_labels = all(
            l == l.lower() and " " not in l
            for l in labels
        )
        checks.append(valid_labels)
    else:
        checks.append(False)

    return (sum(checks) / len(checks)) * 10


def try_parse_json(s):
    """Try to parse a string as JSON. Returns parsed dict/list or None."""
    if not s or not isinstance(s, str):
        return None
    s = s.strip()
    if not (s.startswith("{") or s.startswith("[")):
        return None
    try:
        return json.loads(s)
    except (json.JSONDecodeError, ValueError):
        return None


def score_render_expectation(post, expectations):
    """Score how well a post meets its display_hint's iOS render requirements (0-10)."""
    hints = expectations.get("hints", {})
    hint = post.get("display_hint", "")

    if hint not in hints:
        # Unknown hint — can't validate, give partial credit if basic fields present
        basic = sum([
            bool(post.get("title")),
            bool(post.get("body")),
            bool(post.get("image_url")),
        ])
        return (basic / 3) * 5  # Max 5 for unknown hints

    spec = hints[hint]
    required = spec.get("required_fields", [])
    needs_json = spec.get("external_url_json", False)
    json_keys = spec.get("json_required_keys", [])
    json_nested = spec.get("json_nested", {})

    if not required and not needs_json:
        return 10.0  # No requirements = passes

    checks = []

    # Check required fields
    for field in required:
        val = post.get(field)
        if field == "external_url" and needs_json:
            # Will check JSON separately
            checks.append(bool(val))
        else:
            checks.append(bool(val))

    # Check JSON structure in external_url
    if needs_json:
        ext_url = post.get("external_url", "")
        parsed = try_parse_json(ext_url)
        if parsed and isinstance(parsed, dict):
            checks.append(True)  # Valid JSON
            for k in json_keys:
                checks.append(k in parsed)
            # Check nested keys
            for parent_key, child_keys in json_nested.items():
                parent_val = parsed.get(parent_key)
                if isinstance(parent_val, dict):
                    for ck in child_keys:
                        checks.append(ck in parent_val)
                else:
                    # Parent missing — all children fail
                    for _ in child_keys:
                        checks.append(False)
        else:
            # JSON parse failed — all JSON checks fail
            checks.append(False)
            for _ in json_keys:
                checks.append(False)
            for _, child_keys in json_nested.items():
                for _ in child_keys:
                    checks.append(False)

    if not checks:
        return 10.0
    return (sum(checks) / len(checks)) * 10


def score_content_diversity(posts):
    """Score batch diversity (0-10)."""
    if not posts:
        return 0.0

    checks = []
    n = len(posts)

    # Hint diversity ratio (unique hints / total posts)
    hints = [p.get("display_hint", "unknown") for p in posts]
    hint_ratio = len(set(hints)) / max(n, 1)
    checks.append(min(hint_ratio * 10, 10))

    # At least 2 post types
    types = set(p.get("post_type", p.get("type", "unknown")) for p in posts)
    checks.append(10 if len(types) >= 2 else 5 * len(types))

    # Label concentration <= 60%
    all_labels = []
    for p in posts:
        all_labels.extend(p.get("labels", []) or [])
    if all_labels:
        most_common_count = Counter(all_labels).most_common(1)[0][1]
        concentration = most_common_count / len(all_labels)
        checks.append(10 if concentration <= 0.6 else max(0, 10 - (concentration - 0.6) * 25))
    else:
        checks.append(0)

    # Max 3 consecutive same-type
    max_consecutive = 1
    current_run = 1
    for i in range(1, len(hints)):
        if hints[i] == hints[i - 1]:
            current_run += 1
            max_consecutive = max(max_consecutive, current_run)
        else:
            current_run = 1
    checks.append(10 if max_consecutive <= 3 else max(0, 10 - (max_consecutive - 3) * 2))

    return sum(checks) / len(checks)


def score_non_repetition(posts):
    """Score batch non-repetition (0-10)."""
    if not posts:
        return 0.0

    checks = []
    n = len(posts)

    # Title uniqueness
    titles = [p.get("title", "") for p in posts]
    unique_titles = len(set(t.lower().strip() for t in titles if t))
    checks.append((unique_titles / max(n, 1)) * 10)

    # Body variety (compare first 100 chars)
    bodies = [p.get("body", "")[:100].lower().strip() for p in posts if p.get("body")]
    unique_bodies = len(set(bodies))
    checks.append((unique_bodies / max(len(bodies), 1)) * 10)

    # Label-set variety
    label_sets = [frozenset(p.get("labels", []) or []) for p in posts]
    unique_label_sets = len(set(label_sets))
    checks.append((unique_label_sets / max(n, 1)) * 10)

    return sum(checks) / len(checks)


# ─── LLM Judge Scoring ───

def call_openrouter(model_id, api_key, messages, max_tokens=2000):
    """Call OpenRouter chat completions API. Returns response text."""
    url = "https://openrouter.ai/api/v1/chat/completions"
    payload = json.dumps({
        "model": model_id,
        "messages": messages,
        "max_tokens": max_tokens,
        "temperature": 0.1,
    }).encode("utf-8")

    req = urllib.request.Request(
        url,
        data=payload,
        headers={
            "Content-Type": "application/json",
            "Authorization": f"Bearer {api_key}",
        },
    )

    try:
        with urllib.request.urlopen(req, timeout=120) as resp:
            data = json.loads(resp.read().decode("utf-8"))
            return data["choices"][0]["message"]["content"]
    except (urllib.error.URLError, KeyError, json.JSONDecodeError) as e:
        print(f"  WARNING: OpenRouter call failed: {e}", file=sys.stderr)
        return None


def llm_judge_score(posts, model_key, expectations, compare_model, api_key):
    """Get LLM judge scores for profile_matching, sophistication, image_quality.

    Returns dict with scores (0-10) for each dimension, or None values on failure.
    """
    profile = expectations.get("test_user_profile", {})

    # Build a summary of posts for the judge
    post_summaries = []
    for i, p in enumerate(posts[:20]):  # Cap at 20 to avoid token overflow
        summary = {
            "index": i + 1,
            "title": p.get("title", ""),
            "body": (p.get("body", "") or "")[:200],
            "display_hint": p.get("display_hint", ""),
            "post_type": p.get("post_type", ""),
            "labels": p.get("labels", []),
            "image_url": p.get("image_url", ""),
            "external_url": (p.get("external_url", "") or "")[:500],
            "latitude": p.get("latitude"),
            "longitude": p.get("longitude"),
        }
        post_summaries.append(summary)

    prompt = f"""You are scoring a batch of {len(post_summaries)} social media posts generated by the model "{model_key}" for a personalized feed app.

## Test User Profile
- Location: {profile.get('location', 'Unknown')}
- Interests: {json.dumps(profile.get('interests', []))}
- Family: {json.dumps(profile.get('family', {}))}

## Posts
{json.dumps(post_summaries, indent=2)}

## Score these 3 dimensions (each 0-10, with brief justification):

1. **Profile Matching**: Do posts align with the user's interests (sports, food, music, tech), location (Austin, TX), and family context (partner, kids ages 4 and 7)? Consider interest coverage, local relevance, and family-appropriateness.

2. **Sophistication**: Do posts use rich features? Multi-image, lat/lon coordinates, external_url for articles, structured JSON data in external_url for specialized cards. Higher score = more creative use of the post schema.

3. **Image Quality**: Do posts have images? Are image URLs direct links (not AI generator URLs)? Are they from quality sources (Wikimedia, Unsplash, real URLs)?

Return ONLY a JSON object like:
{{"profile_matching": 7.5, "sophistication": 4.0, "image_quality": 6.5, "reasoning": "brief explanation"}}"""

    messages = [{"role": "user", "content": prompt}]
    response = call_openrouter(compare_model, api_key, messages)

    if not response:
        return {"profile_matching": None, "sophistication": None, "image_quality": None}

    # Extract JSON from response (may be wrapped in markdown code block)
    text = response.strip()
    if "```" in text:
        # Extract from code block
        parts = text.split("```")
        for part in parts:
            part = part.strip()
            if part.startswith("json"):
                part = part[4:].strip()
            if part.startswith("{"):
                text = part
                break

    try:
        scores = json.loads(text)
        return {
            "profile_matching": _clamp(scores.get("profile_matching")),
            "sophistication": _clamp(scores.get("sophistication")),
            "image_quality": _clamp(scores.get("image_quality")),
        }
    except (json.JSONDecodeError, TypeError):
        print(f"  WARNING: Could not parse judge response for {model_key}", file=sys.stderr)
        return {"profile_matching": None, "sophistication": None, "image_quality": None}


def _clamp(val, lo=0, hi=10):
    """Clamp a numeric value to [lo, hi], or return None."""
    if val is None:
        return None
    try:
        return max(lo, min(hi, float(val)))
    except (ValueError, TypeError):
        return None


# ─── Aggregate Scoring ───

def score_model(posts, model_key, expectations, compare_model=None, api_key=None):
    """Compute all dimension scores for a model's posts."""
    if not posts:
        return {dim: 0.0 for dim in WEIGHTS}, 0

    # Per-post deterministic scores
    rule_scores = [score_rule_following(p, expectations) for p in posts]
    render_scores = [score_render_expectation(p, expectations) for p in posts]

    scores = {
        "rule_following": sum(rule_scores) / len(rule_scores),
        "render_expectation": sum(render_scores) / len(render_scores),
        "content_diversity": score_content_diversity(posts),
        "non_repetition": score_non_repetition(posts),
        "profile_matching": None,
        "sophistication": None,
        "image_quality": None,
    }

    # LLM judge scores
    if compare_model and api_key:
        print(f"  Judging {model_key} with {compare_model}...")
        judge = llm_judge_score(posts, model_key, expectations, compare_model, api_key)
        scores["profile_matching"] = judge["profile_matching"]
        scores["sophistication"] = judge["sophistication"]
        scores["image_quality"] = judge["image_quality"]

    return scores, len(posts)


def compute_total(scores):
    """Weighted average of all non-None scores, normalized to 0-10."""
    total_weight = 0
    weighted_sum = 0
    for dim, weight in WEIGHTS.items():
        val = scores.get(dim)
        if val is not None:
            weighted_sum += val * weight
            total_weight += weight
    if total_weight == 0:
        return 0.0
    return weighted_sum / total_weight


# ─── Per-Post Render Scores (for Table 2) ───

def compute_hint_stats(all_model_posts, expectations):
    """Compute per-hint render stats across all models for Table 2."""
    hint_data = defaultdict(lambda: {"models": set(), "render_scores": [], "rule_scores": []})

    for model_key, posts in all_model_posts.items():
        for p in posts:
            hint = p.get("display_hint", "unknown")
            hint_data[hint]["models"].add(model_key)
            hint_data[hint]["render_scores"].append(score_render_expectation(p, expectations))
            hint_data[hint]["rule_scores"].append(score_rule_following(p, expectations))

    return hint_data


# ─── Report Output ───

def print_table1(model_results, total_models):
    """Print Model Comparison table."""
    print("")
    print("=" * 90)
    print("  Table 1 — Model Comparison")
    print("=" * 90)
    print("")

    header = f"{'Model':<16} {'Posts':>5}"
    for dim in ["rule_following", "render_expectation", "profile_matching",
                "content_diversity", "sophistication", "image_quality", "non_repetition"]:
        header += f"  {DIMENSION_LABELS[dim]:>7}"
    header += f"  {'TOTAL':>6}"
    print(header)

    sep = f"{'-' * 16} {'-' * 5}"
    for _ in range(7):
        sep += f"  {'-' * 7}"
    sep += f"  {'-' * 6}"
    print(sep)

    for mr in model_results:
        line = f"{mr['key']:<16} {mr['post_count']:>5}"
        for dim in ["rule_following", "render_expectation", "profile_matching",
                    "content_diversity", "sophistication", "image_quality", "non_repetition"]:
            val = mr["scores"].get(dim)
            if val is None:
                line += f"  {'N/A':>7}"
            else:
                line += f"  {val:>7.1f}"
        line += f"  {mr['total']:>6.1f}"
        print(line)

    print("")


def print_table2(hint_data, total_models):
    """Print Problem Post Types table."""
    print("=" * 90)
    print("  Table 2 — Problem Post Types (hints scoring poorly across models)")
    print("=" * 90)
    print("")

    # Filter to hints with avg render < 7 or attempted by < half of models
    problems = []
    for hint, data in sorted(hint_data.items()):
        avg_render = sum(data["render_scores"]) / len(data["render_scores"]) if data["render_scores"] else 0
        avg_rules = sum(data["rule_scores"]) / len(data["rule_scores"]) if data["rule_scores"] else 0
        model_count = len(data["models"])

        if avg_render < 7.0 or model_count < total_models / 2:
            issue = _diagnose_hint_issue(avg_render, avg_rules, model_count, total_models, hint)
            problems.append({
                "hint": hint,
                "models": f"{model_count}/{total_models}",
                "avg_render": avg_render,
                "avg_rules": avg_rules,
                "issue": issue,
            })

    if not problems:
        print("  No problem hints detected — all hints score well across models.")
        print("")
        return

    header = f"{'Hint':<20} {'Models':>7} {'Avg Render':>10} {'Avg Rules':>10}  {'Likely Issue'}"
    print(header)
    print(f"{'-' * 20} {'-' * 7} {'-' * 10} {'-' * 10}  {'-' * 35}")

    for p in sorted(problems, key=lambda x: x["avg_render"]):
        print(f"{p['hint']:<20} {p['models']:>7} {p['avg_render']:>10.1f} {p['avg_rules']:>10.1f}  {p['issue']}")

    print("")


def _diagnose_hint_issue(avg_render, avg_rules, model_count, total_models, hint):
    """Generate a likely issue description for a problem hint."""
    if model_count == 0:
        return "Never attempted — skill doesn't mention it"
    if avg_render < 3:
        return "JSON schema not produced correctly"
    if avg_render < 5:
        return "Missing required fields for iOS card"
    if model_count < total_models / 2:
        return f"Only {model_count} models attempted this hint"
    if avg_rules < 5:
        return "Rule violations (labels, body length, etc.)"
    return "Partial field coverage"


def print_table3(model_results):
    """Print Problem Areas table."""
    print("=" * 90)
    print("  Table 3 — Problem Areas (dimensions where models struggle)")
    print("=" * 90)
    print("")

    dim_stats = {}
    for dim in WEIGHTS:
        vals = []
        worst = ("", 99)
        best = ("", -1)
        for mr in model_results:
            v = mr["scores"].get(dim)
            if v is not None:
                vals.append(v)
                if v < worst[1]:
                    worst = (mr["key"], v)
                if v > best[1]:
                    best = (mr["key"], v)
        if vals:
            dim_stats[dim] = {
                "avg": sum(vals) / len(vals),
                "worst": worst,
                "best": best,
            }

    header = f"{'Dimension':<22} {'Avg Score':>9}  {'Worst Model':<25} {'Best Model':<25}"
    print(header)
    print(f"{'-' * 22} {'-' * 9}  {'-' * 25} {'-' * 25}")

    for dim in sorted(dim_stats, key=lambda d: dim_stats[d]["avg"]):
        s = dim_stats[dim]
        worst_str = f"{s['worst'][0]} ({s['worst'][1]:.1f})"
        best_str = f"{s['best'][0]} ({s['best'][1]:.1f})"
        print(f"{DIMENSION_LABELS.get(dim, dim):<22} {s['avg']:>9.1f}  {worst_str:<25} {best_str:<25}")

    print("")


# ─── Main ───

def main():
    parser = argparse.ArgumentParser(description="LLM Model Comparison Report")
    parser.add_argument("results_dir", help="Directory containing per-model JSON files")
    parser.add_argument("models_conf", help="Path to models.conf")
    parser.add_argument("--expectations", required=True, help="Path to expectations.json")
    parser.add_argument("--compare-model", help="OpenRouter model ID for LLM judge scoring")
    parser.add_argument("--openrouter-key", help="OpenRouter API key")
    args = parser.parse_args()

    models = parse_models_conf(args.models_conf)
    expectations = load_expectations(args.expectations)

    print("")
    print("=" * 90)
    print("       LLM Model Comparison Report — Quality Scoring")
    print("=" * 90)

    # Score each model
    model_results = []
    all_model_posts = {}

    for m in models:
        key = m["key"]
        posts = load_results(args.results_dir, key)
        if not posts:
            print(f"  {key}: no results found, skipping")
            continue

        all_model_posts[key] = posts
        scores, post_count = score_model(
            posts, key, expectations,
            compare_model=args.compare_model,
            api_key=args.openrouter_key,
        )
        total = compute_total(scores)
        model_results.append({
            "key": key,
            "display_name": m["display_name"],
            "post_count": post_count,
            "scores": scores,
            "total": total,
        })

    if not model_results:
        print("\n  No results found in any model's JSON files.")
        print(f"  Expected files in: {args.results_dir}/<model_key>.json")
        sys.exit(1)

    # Sort by total score descending
    model_results.sort(key=lambda x: x["total"], reverse=True)

    # Table 1: Model Comparison
    print_table1(model_results, len(models))

    # Table 2: Problem Post Types
    hint_data = compute_hint_stats(all_model_posts, expectations)
    print_table2(hint_data, len(all_model_posts))

    # Table 3: Problem Areas
    print_table3(model_results)


if __name__ == "__main__":
    main()
