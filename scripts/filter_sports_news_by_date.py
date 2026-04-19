#!/usr/bin/env python3
"""Filter sports-news candidates by publication recency.

Input: JSON array from stdin with items like:
[
  {"title": "...", "url": "...", "published_at": "2026-04-18T09:14:00Z"}
]

Output JSON:
{
  "today": "2026-04-18",
  "timezone": "America/Vancouver",
  "max_age_days": 1,
  "fresh": [...],
  "stale": [...],
  "invalid": [...]
}

Rules:
- publication date must parse as ISO-8601 timestamp or YYYY-MM-DD
- item is fresh if 0 <= (today_local - published_local_date) <= max_age_days
- future-dated items are marked stale (negative age)
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import sys
from dataclasses import dataclass
from zoneinfo import ZoneInfo


@dataclass
class ParsedDate:
    original: str
    local_date: dt.date


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="Filter sports news by publication date")
    p.add_argument("--timezone", default="America/Vancouver", help="IANA timezone (default: America/Vancouver)")
    p.add_argument("--max-age-days", type=int, default=1, help="Max age in days to accept (default: 1)")
    p.add_argument("--today", help="Override today's date (YYYY-MM-DD) for deterministic testing")
    return p.parse_args()


def parse_iso_like(raw: str, tz: ZoneInfo) -> ParsedDate:
    s = raw.strip()
    if not s:
        raise ValueError("empty date")

    # Normalize trailing Z for fromisoformat compatibility
    normalized = s[:-1] + "+00:00" if s.endswith("Z") else s

    # Try full datetime first
    try:
        parsed_dt = dt.datetime.fromisoformat(normalized)
        if parsed_dt.tzinfo is None:
            parsed_dt = parsed_dt.replace(tzinfo=tz)
        local_date = parsed_dt.astimezone(tz).date()
        return ParsedDate(original=raw, local_date=local_date)
    except ValueError:
        pass

    # Fallback to date-only
    try:
        parsed_date = dt.date.fromisoformat(s)
        return ParsedDate(original=raw, local_date=parsed_date)
    except ValueError as exc:
        raise ValueError(f"unsupported date format: {raw}") from exc


def main() -> int:
    args = parse_args()

    if args.max_age_days < 0:
        print(json.dumps({"error": "--max-age-days must be >= 0"}))
        return 2

    tz = ZoneInfo(args.timezone)
    if args.today:
        today = dt.date.fromisoformat(args.today)
    else:
        today = dt.datetime.now(tz).date()

    try:
        payload = json.load(sys.stdin)
    except json.JSONDecodeError as exc:
        print(json.dumps({"error": f"invalid JSON: {exc}"}))
        return 2

    if not isinstance(payload, list):
        print(json.dumps({"error": "input must be a JSON array"}))
        return 2

    fresh: list[dict] = []
    stale: list[dict] = []
    invalid: list[dict] = []

    for item in payload:
        if not isinstance(item, dict):
            invalid.append({"item": item, "reason": "item is not an object"})
            continue

        published = item.get("published_at") or item.get("publishedAt") or item.get("date")
        if not isinstance(published, str):
            invalid.append({"item": item, "reason": "missing published_at/publishedAt/date string"})
            continue

        try:
            parsed = parse_iso_like(published, tz)
        except ValueError as exc:
            invalid.append({"item": item, "reason": str(exc)})
            continue

        age_days = (today - parsed.local_date).days
        enriched = {
            **item,
            "published_local_date": parsed.local_date.isoformat(),
            "age_days": age_days,
        }

        if 0 <= age_days <= args.max_age_days:
            fresh.append(enriched)
        else:
            stale.append(enriched)

    result = {
        "today": today.isoformat(),
        "timezone": args.timezone,
        "max_age_days": args.max_age_days,
        "fresh": fresh,
        "stale": stale,
        "invalid": invalid,
    }
    print(json.dumps(result, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
