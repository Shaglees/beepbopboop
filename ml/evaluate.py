"""
Offline evaluation metrics for the two-tower model.

Metrics:
    AUC-ROC   — area under ROC curve; target > 0.75 on held-out data
    NDCG@k    — normalised discounted cumulative gain at top-k
    Precision@k — fraction of top-k ranked items that were actually engaged with

Usage:
    python evaluate.py --checkpoint ranker.pt --data data/test_pairs.parquet
"""
from __future__ import annotations

import argparse
import math
from typing import Sequence

import torch
from torch.utils.data import DataLoader


def auc_roc(labels: Sequence[int], scores: Sequence[float]) -> float:
    """Compute AUC-ROC from binary labels and real-valued scores."""
    from sklearn.metrics import roc_auc_score
    return float(roc_auc_score(labels, scores))


def ndcg_at_k(
    model: "TwoTowerModel",
    user_vecs: torch.Tensor,
    post_vecs: torch.Tensor,
    labels: torch.Tensor,
    k: int = 10,
) -> float:
    """
    Compute NDCG@k averaged over users.

    For each user, rank post candidates by model score and measure how well
    that ranking recovers the ground-truth engagement order.
    """
    model.eval()
    with torch.no_grad():
        scores = model(user_vecs, post_vecs).squeeze(1).tolist()
    labels_list = labels.squeeze(1).tolist()

    ranked = sorted(zip(scores, labels_list), reverse=True)
    ideal = sorted(labels_list, reverse=True)

    def dcg(items, k):
        return sum(rel / math.log2(i + 2) for i, (_, rel) in enumerate(items[:k]))

    actual_dcg = dcg(ranked, k)
    ideal_dcg = sum(rel / math.log2(i + 2) for i, rel in enumerate(ideal[:k]))
    return actual_dcg / ideal_dcg if ideal_dcg > 0 else 0.0


def precision_at_k(
    scores: Sequence[float],
    labels: Sequence[int],
    k: int = 10,
    threshold: float = 0.5,
) -> float:
    """Fraction of top-k scored items with label >= threshold."""
    ranked = sorted(zip(scores, labels), reverse=True)
    top_k = ranked[:k]
    hits = sum(1 for _, lbl in top_k if lbl >= threshold)
    return hits / k if k > 0 else 0.0


def main():
    parser = argparse.ArgumentParser(description="Evaluate two-tower model")
    parser.add_argument("--checkpoint", required=True)
    parser.add_argument("--data", required=True)
    parser.add_argument("--k", type=int, default=10)
    args = parser.parse_args()

    from model import TwoTowerModel
    from dataset import load_training_pairs, PairDataset

    ckpt = torch.load(args.checkpoint, map_location="cpu")
    model = TwoTowerModel(
        input_dim=ckpt["input_dim"],
        hidden=ckpt["hidden"],
        repr_dim=ckpt["repr_dim"],
    )
    model.load_state_dict(ckpt["model_state_dict"])
    model.eval()

    pairs = load_training_pairs(args.data)
    dataset = PairDataset(pairs)
    loader = DataLoader(dataset, batch_size=512)

    all_scores, all_labels = [], []
    with torch.no_grad():
        for u, p, lbl in loader:
            s = model(u, p).squeeze(1)
            all_scores.extend(s.tolist())
            all_labels.extend(lbl.squeeze(1).tolist())

    binary = [1 if l >= 0.5 else 0 for l in all_labels]
    auc = auc_roc(binary, all_scores)
    prec = precision_at_k(all_scores, binary, k=args.k)
    print(f"AUC-ROC:        {auc:.4f}")
    print(f"Precision@{args.k}:   {prec:.4f}")


if __name__ == "__main__":
    main()
