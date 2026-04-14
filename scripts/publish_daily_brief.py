#!/usr/bin/env python3
"""
Publish a daily brief section post to BeepBopBoop, with idempotency by title.

Usage example:
  python3 scripts/publish_daily_brief.py \
    --title "Daily Brief — Calendar — 2026-04-14" \
    --body "09:00 Standup..." \
    --labels daily-brief,calendar \
    --visibility private
"""

from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path
from typing import Dict, List
from urllib import error, request

CONFIG_PATH = Path.home() / ".config" / "beepbopboop" / "config"


def parse_config(path: Path) -> Dict[str, str]:
    if not path.exists():
        raise FileNotFoundError(f"Config not found: {path}")
    out: Dict[str, str] = {}
    for raw in path.read_text().splitlines():
        line = raw.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        k, v = line.split("=", 1)
        out[k.strip()] = v.strip()
    return out


def http_json(method: str, url: str, token: str, payload: dict | None = None):
    data = None
    if payload is not None:
        data = json.dumps(payload).encode("utf-8")
    req = request.Request(url=url, method=method, data=data)
    req.add_header("Authorization", f"Bearer {token}")
    req.add_header("Content-Type", "application/json")
    with request.urlopen(req, timeout=20) as resp:
        body = resp.read().decode("utf-8")
        return resp.status, json.loads(body) if body else {}


def load_recent_titles(api_url: str, token: str, limit: int = 100) -> List[str]:
    status, data = http_json("GET", f"{api_url.rstrip('/')}/posts?limit={limit}", token)
    if status != 200 or not isinstance(data, list):
        return []
    return [str(p.get("title", "")) for p in data if isinstance(p, dict)]


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--title", required=True)
    ap.add_argument("--body", required=True)
    ap.add_argument("--labels", default="daily-brief")
    ap.add_argument("--visibility", default="private", choices=["public", "personal", "private"])
    ap.add_argument("--post-type", default="article", choices=["event", "place", "discovery", "article", "video"])
    ap.add_argument("--locality", default="")
    ap.add_argument("--external-url", default="")
    ap.add_argument("--image-url", default="")
    ap.add_argument("--dry-run", action="store_true")
    args = ap.parse_args()

    cfg = parse_config(CONFIG_PATH)
    api_url = cfg.get("BEEPBOPBOOP_API_URL", "").strip()
    token = cfg.get("BEEPBOPBOOP_AGENT_TOKEN", "").strip()
    if not api_url or not token:
        print("Missing BEEPBOPBOOP_API_URL or BEEPBOPBOOP_AGENT_TOKEN in config", file=sys.stderr)
        return 2

    labels = [x.strip() for x in args.labels.split(",") if x.strip()]
    payload = {
        "title": args.title,
        "body": args.body,
        "post_type": args.post_type,
        "visibility": args.visibility,
        "labels": labels,
        "locality": args.locality,
        "external_url": args.external_url,
        "image_url": args.image_url,
    }

    # Idempotency: skip if same title already exists in recent posts.
    try:
        recent_titles = load_recent_titles(api_url, token, limit=100)
    except Exception as exc:
        print(f"WARN: could not load recent posts for dedupe: {exc}", file=sys.stderr)
        recent_titles = []

    if args.title in recent_titles:
        print(json.dumps({"status": "skipped", "reason": "title_exists", "title": args.title}))
        return 0

    if args.dry_run:
        print(json.dumps({"status": "dry_run", "payload": payload}, ensure_ascii=False))
        return 0

    try:
        status, data = http_json("POST", f"{api_url.rstrip('/')}/posts", token, payload)
    except error.HTTPError as exc:
        msg = exc.read().decode("utf-8", errors="ignore")
        print(json.dumps({"status": "error", "http_status": exc.code, "error": msg}), file=sys.stderr)
        return 1
    except Exception as exc:
        print(json.dumps({"status": "error", "error": str(exc)}), file=sys.stderr)
        return 1

    if status != 201:
        print(json.dumps({"status": "error", "http_status": status, "response": data}), file=sys.stderr)
        return 1

    print(json.dumps({"status": "created", "id": data.get("id"), "title": data.get("title")}, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
