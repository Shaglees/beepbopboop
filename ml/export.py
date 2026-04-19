"""
Export trained two-tower model weights for Go inference.

Primary format: JSON checkpoint (used by Go Ranker — no native deps required).
Optional: ONNX export for full model fidelity (requires onnx + onnxruntime).

Usage:
    # JSON export (default — recommended for Go integration):
    python export.py --checkpoint ranker.pt --output ranker.json

    # ONNX export (optional):
    python export.py --checkpoint ranker.pt --output ranker.onnx --format onnx
"""
from __future__ import annotations

import argparse
import json
import os


def export_json(model: "TwoTowerModel", path: str) -> None:
    """
    Export the first linear layer weights from each tower to a JSON checkpoint.

    The Go Ranker loads this file as its scoring model.  The single-layer
    approximation trades a small amount of accuracy for simplicity — upgrade
    to ONNX for full fidelity once the pipeline is proven.
    """
    weights = model.export_weights()
    with open(path, "w") as f:
        json.dump(weights, f)
    size_kb = os.path.getsize(path) / 1024
    print(f"JSON checkpoint saved to {path} ({size_kb:.1f} KB)")


def load_json_ranker(path: str) -> "TwoTowerModel":
    """
    Load a model whose first-layer weights were serialised to JSON.
    Useful for round-trip testing: export → load → compare outputs.
    """
    import torch
    import numpy as np
    from model import TwoTowerModel

    with open(path) as f:
        ckpt = json.load(f)

    input_dim = ckpt["input_dim"]
    repr_dim = ckpt["repr_dim"]

    # Build a model with repr_dim as both hidden and output (no second layer)
    model = TwoTowerModel(input_dim=input_dim, hidden=repr_dim, repr_dim=repr_dim)

    uw = torch.tensor(ckpt["user_weights"], dtype=torch.float32)
    pw = torch.tensor(ckpt["post_weights"], dtype=torch.float32)

    with torch.no_grad():
        model.user_tower.net[0].weight.copy_(uw)
        model.post_tower.net[0].weight.copy_(pw)

    return model


def export_onnx(model: "TwoTowerModel", path: str) -> None:
    """
    Export the full two-tower model to ONNX format.
    Requires: pip install onnx onnxruntime
    """
    import torch

    model.eval()
    dim = model.input_dim
    dummy_user = torch.randn(1, dim)
    dummy_post = torch.randn(1, dim)

    torch.onnx.export(
        model,
        (dummy_user, dummy_post),
        path,
        input_names=["user_vec", "post_vec"],
        output_names=["score"],
        dynamic_axes={
            "user_vec": {0: "batch"},
            "post_vec": {0: "batch"},
            "score": {0: "batch"},
        },
        opset_version=17,
    )
    size_mb = os.path.getsize(path) / (1024 * 1024)
    print(f"ONNX model saved to {path} ({size_mb:.2f} MB)")


def main():
    parser = argparse.ArgumentParser(description="Export two-tower model")
    parser.add_argument("--checkpoint", required=True, help="Path to ranker.pt")
    parser.add_argument("--output", required=True, help="Output file path")
    parser.add_argument("--format", choices=["json", "onnx"], default="json")
    args = parser.parse_args()

    import torch
    from model import TwoTowerModel

    ckpt = torch.load(args.checkpoint, map_location="cpu")
    model = TwoTowerModel(
        input_dim=ckpt["input_dim"],
        hidden=ckpt["hidden"],
        repr_dim=ckpt["repr_dim"],
    )
    model.load_state_dict(ckpt["model_state_dict"])
    model.eval()

    if args.format == "json":
        export_json(model, args.output)
    else:
        export_onnx(model, args.output)


if __name__ == "__main__":
    main()
