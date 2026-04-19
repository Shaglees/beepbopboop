"""
Training loop for the two-tower ranking model.

Quick start (synthetic data):
    python train.py --synthetic --n_pairs 10000 --epochs 50

Production (real data):
    python train.py --data data/training_pairs.parquet --epochs 100 --output ranker.pt

Then export for Go:
    python export.py --checkpoint ranker.pt --output ranker.json
"""
from __future__ import annotations

import argparse
import time
from typing import Optional

import torch
import torch.nn as nn
from torch.utils.data import DataLoader, random_split


# ---------------------------------------------------------------------------
# public helpers (used by test_model.py)
# ---------------------------------------------------------------------------

def train_synthetic(
    model: "TwoTowerModel",
    n_pairs: int = 10_000,
    epochs: int = 50,
    lr: float = 1e-3,
    batch_size: int = 256,
    seed: int = 42,
) -> dict:
    """
    Train model on synthetic pairs. Returns a dict with final train/val loss.
    Useful for unit tests that need a minimally-trained model.
    """
    from dataset import make_synthetic_pairs, PairDataset

    torch.manual_seed(seed)
    pairs = make_synthetic_pairs(n_pairs, input_dim=model.input_dim, seed=seed)
    dataset = PairDataset(pairs)

    val_size = max(1, len(dataset) // 10)
    train_size = len(dataset) - val_size
    train_ds, val_ds = random_split(dataset, [train_size, val_size])

    train_loader = DataLoader(train_ds, batch_size=batch_size, shuffle=True)
    val_loader = DataLoader(val_ds, batch_size=batch_size)

    return _run_training(model, train_loader, val_loader, epochs=epochs, lr=lr)


def evaluate_auc(
    model: "TwoTowerModel",
    n_pairs: int = 500,
    input_dim: int = 1536,
    seed: int = 99,
) -> float:
    """Compute AUC-ROC on a fresh synthetic dataset (no overlap with training)."""
    from dataset import make_synthetic_pairs, PairDataset
    from sklearn.metrics import roc_auc_score

    pairs = make_synthetic_pairs(n_pairs, input_dim=input_dim, seed=seed)
    dataset = PairDataset(pairs)
    loader = DataLoader(dataset, batch_size=256)

    model.eval()
    all_scores, all_labels = [], []
    with torch.no_grad():
        for u, p, lbl in loader:
            scores = model(u, p).squeeze(1)
            all_scores.extend(scores.tolist())
            all_labels.extend(lbl.squeeze(1).tolist())

    binary_labels = [1 if l >= 0.5 else 0 for l in all_labels]
    if len(set(binary_labels)) < 2:
        return 0.5
    return float(roc_auc_score(binary_labels, all_scores))


# ---------------------------------------------------------------------------
# core training loop
# ---------------------------------------------------------------------------

def _run_training(
    model: "TwoTowerModel",
    train_loader: DataLoader,
    val_loader: DataLoader,
    epochs: int = 50,
    lr: float = 1e-3,
    patience: int = 10,
) -> dict:
    criterion = nn.BCELoss()
    optimizer = torch.optim.Adam(model.parameters(), lr=lr)
    scheduler = torch.optim.lr_scheduler.CosineAnnealingLR(optimizer, T_max=epochs)

    best_val_loss = float("inf")
    epochs_no_improve = 0
    history = {"train_loss": [], "val_loss": []}

    for epoch in range(epochs):
        model.train()
        train_loss = _epoch_loss(model, train_loader, criterion, optimizer)

        model.eval()
        with torch.no_grad():
            val_loss = _epoch_loss(model, val_loader, criterion)

        scheduler.step()
        history["train_loss"].append(train_loss)
        history["val_loss"].append(val_loss)

        if val_loss < best_val_loss - 1e-4:
            best_val_loss = val_loss
            epochs_no_improve = 0
        else:
            epochs_no_improve += 1
            if epochs_no_improve >= patience:
                break

    return history


def _epoch_loss(
    model: "TwoTowerModel",
    loader: DataLoader,
    criterion: nn.Module,
    optimizer: Optional[torch.optim.Optimizer] = None,
) -> float:
    total_loss, n_batches = 0.0, 0
    for user_vecs, post_vecs, labels in loader:
        scores = model(user_vecs, post_vecs)
        # Clip labels to [0, 1] for BCE (hard negatives stored as -0.5)
        labels_clamped = labels.clamp(0.0, 1.0)
        loss = criterion(scores, labels_clamped)
        if optimizer is not None:
            optimizer.zero_grad()
            loss.backward()
            optimizer.step()
        total_loss += loss.item()
        n_batches += 1
    return total_loss / max(n_batches, 1)


# ---------------------------------------------------------------------------
# CLI entry point
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(description="Train two-tower ranking model")
    parser.add_argument("--data", help="Path to training_pairs.parquet")
    parser.add_argument("--synthetic", action="store_true",
                        help="Use synthetic data instead of --data")
    parser.add_argument("--n_pairs", type=int, default=10_000)
    parser.add_argument("--epochs", type=int, default=50)
    parser.add_argument("--lr", type=float, default=1e-3)
    parser.add_argument("--batch_size", type=int, default=256)
    parser.add_argument("--input_dim", type=int, default=1536)
    parser.add_argument("--hidden", type=int, default=256)
    parser.add_argument("--repr_dim", type=int, default=128)
    parser.add_argument("--output", default="ranker.pt")
    args = parser.parse_args()

    from model import TwoTowerModel

    model = TwoTowerModel(
        input_dim=args.input_dim,
        hidden=args.hidden,
        repr_dim=args.repr_dim,
    )

    if args.synthetic:
        print(f"Training on {args.n_pairs} synthetic pairs for {args.epochs} epochs …")
        history = train_synthetic(model, n_pairs=args.n_pairs, epochs=args.epochs,
                                   lr=args.lr, batch_size=args.batch_size)
    elif args.data:
        from dataset import load_training_pairs, PairDataset
        print(f"Loading data from {args.data} …")
        pairs = load_training_pairs(args.data)
        dataset = PairDataset(pairs)
        val_size = max(1, len(dataset) // 10)
        train_size = len(dataset) - val_size
        train_ds, val_ds = random_split(dataset, [train_size, val_size])
        train_loader = DataLoader(train_ds, batch_size=args.batch_size, shuffle=True)
        val_loader = DataLoader(val_ds, batch_size=args.batch_size)
        history = _run_training(model, train_loader, val_loader,
                                epochs=args.epochs, lr=args.lr)
    else:
        parser.error("Provide --data or --synthetic")

    final_val = history["val_loss"][-1]
    print(f"Final val loss: {final_val:.4f}")

    torch.save({"model_state_dict": model.state_dict(),
                "input_dim": args.input_dim,
                "hidden": args.hidden,
                "repr_dim": args.repr_dim}, args.output)
    print(f"Checkpoint saved to {args.output}")

    auc = evaluate_auc(model, input_dim=args.input_dim)
    print(f"Val AUC-ROC (synthetic): {auc:.4f}")


if __name__ == "__main__":
    main()
