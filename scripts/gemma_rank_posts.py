#!/usr/bin/env python3
"""
Rank BeepBopBoop candidate posts with local Gemma (Ollama).

Input: JSON array of candidates via stdin or --input file.
Each candidate should include: title, body, post_type, labels (optional).

Output: JSON array sorted by score descending with added fields:
- score (0-100)
- novelty_score (0-100)
- utility_score (0-100)
- quality_score (0-100)
- ml_context_tags (list[str])
- rank_reason

Example:
  python3 scripts/gemma_rank_posts.py --model gemma4:e2b --input /tmp/candidates.json
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any
from urllib import request, error


def load_candidates(path: str | None) -> list[dict[str, Any]]:
    if path:
        data = Path(path).read_text()
    else:
        data = sys.stdin.read()
    parsed = json.loads(data)
    if not isinstance(parsed, list):
        raise ValueError("Input must be a JSON array of candidate posts")
    out: list[dict[str, Any]] = []
    for i, item in enumerate(parsed):
        if not isinstance(item, dict):
            continue
        title = str(item.get("title", "")).strip()
        body = str(item.get("body", "")).strip()
        if not title or not body:
            continue
        out.append(
            {
                "id": item.get("id", f"cand-{i+1}"),
                "title": title,
                "body": body,
                "post_type": str(item.get("post_type", "discovery")),
                "labels": item.get("labels", []) if isinstance(item.get("labels", []), list) else [],
                "locality": str(item.get("locality", "")),
                "external_url": str(item.get("external_url", "")),
            }
        )
    return out


def call_ollama(prompt: str, model: str, host: str, timeout: int) -> dict[str, Any]:
    payload = {
        "model": model,
        "prompt": prompt,
        "stream": False,
        "format": "json",
        "options": {
            "temperature": 0.1,
            "num_predict": 700,
        },
    }
    req = request.Request(
        url=f"{host.rstrip('/')}/api/generate",
        data=json.dumps(payload).encode("utf-8"),
        method="POST",
        headers={"Content-Type": "application/json"},
    )
    with request.urlopen(req, timeout=timeout) as resp:
        body = resp.read().decode("utf-8")
        outer = json.loads(body)
    txt = outer.get("response", "{}")
    parsed = json.loads(txt)
    if not isinstance(parsed, dict):
        raise ValueError("Gemma response is not JSON object")
    return parsed


def make_prompt(candidates: list[dict[str, Any]]) -> str:
    schema = {
        "ranked": [
            {
                "id": "candidate id",
                "score": "0-100",
                "novelty_score": "0-100",
                "utility_score": "0-100",
                "quality_score": "0-100",
                "ml_context_tags": ["short-tags-for-feed-learning"],
                "rank_reason": "short reason",
            }
        ]
    }
    return (
        "You are ranking social discovery posts for a personalized feed.\n"
        "Return strict JSON only. No markdown.\n"
        "Scoring goals:\n"
        "- novelty_score: freshness/non-repetition\n"
        "- utility_score: actionable value, specifics (time/place/price/link)\n"
        "- quality_score: clarity and hook quality\n"
        "- score: weighted overall (novelty 35%, utility 35%, quality 30%)\n"
        "Also generate ml_context_tags to improve ranking (topic, intent, context, audience-safe tags).\n"
        "Use compact lowercase hyphen tags. 4-8 tags each.\n"
        f"Output schema: {json.dumps(schema)}\n"
        f"Candidates: {json.dumps(candidates, ensure_ascii=False)}"
    )


def fallback_rank(candidates: list[dict[str, Any]]) -> dict[str, Any]:
    ranked = []
    for c in candidates:
        body = c["body"]
        title = c["title"]
        utility = min(100, 30 + (10 if "http" in body else 0) + (10 if "$" in body else 0) + (10 if any(x in body.lower() for x in ["open", "hours", "today", "tonight"]) else 0))
        quality = min(100, 50 + (10 if len(title) <= 80 else 0) + (10 if len(body) <= 280 else 0))
        novelty = 60
        score = round(0.35 * novelty + 0.35 * utility + 0.30 * quality)
        ranked.append(
            {
                "id": c["id"],
                "score": score,
                "novelty_score": novelty,
                "utility_score": utility,
                "quality_score": quality,
                "ml_context_tags": ["fallback-ranker", c.get("post_type", "discovery")],
                "rank_reason": "fallback heuristic rank",
            }
        )
    ranked.sort(key=lambda x: x["score"], reverse=True)
    return {"ranked": ranked}


def merge(candidates: list[dict[str, Any]], ranked: dict[str, Any]) -> list[dict[str, Any]]:
    by_id = {c["id"]: c for c in candidates}
    out = []
    for r in ranked.get("ranked", []):
        cid = r.get("id")
        if cid not in by_id:
            continue
        c = dict(by_id[cid])
        c["score"] = int(r.get("score", 0))
        c["novelty_score"] = int(r.get("novelty_score", 0))
        c["utility_score"] = int(r.get("utility_score", 0))
        c["quality_score"] = int(r.get("quality_score", 0))
        tags = r.get("ml_context_tags", [])
        c["ml_context_tags"] = [str(t).strip().lower() for t in tags if str(t).strip()]
        c["rank_reason"] = str(r.get("rank_reason", ""))
        out.append(c)
    # add any missing candidates at end
    seen = {x["id"] for x in out}
    for c in candidates:
        if c["id"] not in seen:
            c2 = dict(c)
            c2.update({
                "score": 0,
                "novelty_score": 0,
                "utility_score": 0,
                "quality_score": 0,
                "ml_context_tags": ["unranked"],
                "rank_reason": "missing from model output",
            })
            out.append(c2)
    out.sort(key=lambda x: x["score"], reverse=True)
    return out


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--input", help="Path to candidates JSON array. If omitted, read stdin")
    ap.add_argument("--model", default="gemma4:e2b")
    ap.add_argument("--host", default="http://localhost:11434")
    ap.add_argument("--timeout", type=int, default=180)
    args = ap.parse_args()

    candidates = load_candidates(args.input)
    if not candidates:
        print("[]")
        return 0

    prompt = make_prompt(candidates)

    try:
        ranked = call_ollama(prompt=prompt, model=args.model, host=args.host, timeout=args.timeout)
    except Exception:
        ranked = fallback_rank(candidates)

    merged = merge(candidates, ranked)
    print(json.dumps(merged, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
