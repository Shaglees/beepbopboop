#!/usr/bin/env python3
"""
Evaluate retrieval quality for two embedding model versions using label-overlap@K.

Usage:
  python3 scripts/evaluate_embedding_retrieval.py \
    --db-url "$DATABASE_URL" \
    --model-a "hash-v1" \
    --model-b "google/gemini-embedding-002:dim1536" \
    --sample-size 200 --k 10
"""

import argparse
import json
import math
import random
from typing import Dict, List, Tuple

import psycopg


def parse_vec(v) -> List[float]:
    if isinstance(v, list):
        return [float(x) for x in v]
    return []


def cosine(a: List[float], b: List[float]) -> float:
    if not a or not b or len(a) != len(b):
        return -1.0
    dot = sum(x * y for x, y in zip(a, b))
    na = math.sqrt(sum(x * x for x in a))
    nb = math.sqrt(sum(y * y for y in b))
    if na == 0 or nb == 0:
        return -1.0
    return dot / (na * nb)


def avg_overlap_at_k(rows: List[Tuple[str, List[str], List[float]]], k: int) -> float:
    if len(rows) < 2:
        return 0.0
    scores = []
    for i, (pid, labels, vec) in enumerate(rows):
        if not labels:
            continue
        sims = []
        for j, (pid2, labels2, vec2) in enumerate(rows):
            if i == j:
                continue
            sims.append((cosine(vec, vec2), labels2))
        sims.sort(key=lambda x: x[0], reverse=True)
        top = sims[:k]
        if not top:
            continue
        overlap = 0.0
        qset = set(labels)
        for _, ls in top:
            overlap += len(qset.intersection(ls)) / max(len(qset.union(ls)), 1)
        scores.append(overlap / len(top))
    return sum(scores) / len(scores) if scores else 0.0


def load_rows(conn, model_version: str, sample_size: int):
    q = """
    SELECT p.id,
           COALESCE(p.labels, '[]'::jsonb)::text AS labels_json,
           pe.embedding
    FROM post_embeddings pe
    JOIN posts p ON p.id = pe.post_id
    WHERE pe.model_version = %s
      AND p.status = 'published'
      AND jsonb_array_length(COALESCE(p.labels, '[]'::jsonb)) > 0
    ORDER BY p.created_at DESC
    LIMIT %s
    """
    out = []
    with conn.cursor() as cur:
        cur.execute(q, (model_version, sample_size * 3))
        for pid, labels_json, emb in cur.fetchall():
            labels = json.loads(labels_json)
            if not isinstance(labels, list):
                continue
            out.append((pid, [str(x) for x in labels], parse_vec(emb)))
    random.shuffle(out)
    return out[:sample_size]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--db-url", required=True)
    ap.add_argument("--model-a", required=True)
    ap.add_argument("--model-b", required=True)
    ap.add_argument("--sample-size", type=int, default=200)
    ap.add_argument("--k", type=int, default=10)
    args = ap.parse_args()

    with psycopg.connect(args.db_url) as conn:
        rows_a = load_rows(conn, args.model_a, args.sample_size)
        rows_b = load_rows(conn, args.model_b, args.sample_size)

    score_a = avg_overlap_at_k(rows_a, args.k)
    score_b = avg_overlap_at_k(rows_b, args.k)

    print(json.dumps({
        "model_a": args.model_a,
        "model_b": args.model_b,
        "sample_size_a": len(rows_a),
        "sample_size_b": len(rows_b),
        "k": args.k,
        "label_overlap_at_k": {
            "model_a": score_a,
            "model_b": score_b,
            "delta_b_minus_a": score_b - score_a,
        },
    }, indent=2))


if __name__ == "__main__":
    main()
