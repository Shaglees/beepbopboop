"""
Dataset utilities for the two-tower training pipeline.

Production use:
    from dataset import load_training_pairs
    pairs = load_training_pairs("data/training_pairs.parquet")

Label conventions (from issue #42):
    1.0   positive  — dwell_ms >= 10 000ms OR save OR reaction='more'
    0.7   weak pos  — dwell_ms 3 000–10 000ms OR click
    0.0   implicit neg — view with dwell_ms < 3 000ms, no interaction
   -0.5   hard neg  — reaction='less' OR reaction='not_for_me'
          (clipped to 0.0 for BCE; used as negative anchor in margin loss)
"""
from __future__ import annotations

from dataclasses import dataclass
from typing import Optional

import numpy as np
import torch
from torch.utils.data import Dataset


@dataclass
class TrainingPair:
    user_vec: np.ndarray   # (input_dim,)
    post_vec: np.ndarray   # (input_dim,)
    label: float           # see conventions above


class PairDataset(Dataset):
    def __init__(self, pairs: list[TrainingPair]):
        self.user_vecs = torch.tensor(
            np.stack([p.user_vec for p in pairs]), dtype=torch.float32
        )
        self.post_vecs = torch.tensor(
            np.stack([p.post_vec for p in pairs]), dtype=torch.float32
        )
        self.labels = torch.tensor(
            [p.label for p in pairs], dtype=torch.float32
        ).unsqueeze(1)

    def __len__(self) -> int:
        return len(self.labels)

    def __getitem__(self, idx: int):
        return self.user_vecs[idx], self.post_vecs[idx], self.labels[idx]


def make_synthetic_pairs(
    n_pairs: int,
    input_dim: int = 1536,
    positive_ratio: float = 0.5,
    seed: int = 42,
) -> list[TrainingPair]:
    """
    Generate synthetic (user_vec, post_vec, label) triples for unit-testing
    the training pipeline without a real database connection.

    Positive pairs: post_vec is a noisy copy of user_vec (cosine sim ≈ 0.95).
    Negative pairs: post_vec is an independent random vector.
    """
    rng = np.random.default_rng(seed)
    pairs = []
    n_pos = int(n_pairs * positive_ratio)
    n_neg = n_pairs - n_pos

    def unit(v: np.ndarray) -> np.ndarray:
        n = np.linalg.norm(v)
        return v / n if n > 1e-10 else v

    for _ in range(n_pos):
        u = unit(rng.standard_normal(input_dim).astype(np.float32))
        p = unit(u + 0.05 * rng.standard_normal(input_dim).astype(np.float32))
        pairs.append(TrainingPair(u, p, 1.0))

    for _ in range(n_neg):
        u = unit(rng.standard_normal(input_dim).astype(np.float32))
        p = unit(rng.standard_normal(input_dim).astype(np.float32))
        pairs.append(TrainingPair(u, p, 0.0))

    rng.shuffle(pairs)
    return pairs


def load_training_pairs(path: str) -> list[TrainingPair]:
    """Load a Parquet file produced by the data-export pipeline."""
    try:
        import pandas as pd
    except ImportError:
        raise ImportError("pandas is required to load parquet files: pip install pandas pyarrow")

    df = pd.read_parquet(path)
    pairs = []
    for _, row in df.iterrows():
        pairs.append(TrainingPair(
            user_vec=np.array(row["user_embedding"], dtype=np.float32),
            post_vec=np.array(row["post_embedding"], dtype=np.float32),
            label=float(row["label"]),
        ))
    return pairs
