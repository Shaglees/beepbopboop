"""
Two-tower ranking model.

Architecture:
  user_vec (input_dim) ──► Linear(hidden) ──► ReLU ──► Linear(repr_dim) ──► L2Norm
  post_vec (input_dim) ──► Linear(hidden) ──► ReLU ──► Linear(repr_dim) ──► L2Norm
                                                                 └──────┬──────┘
                                                                  dot product
                                                                 (x + 1) / 2
                                                               relevance ∈ [0, 1]

Usage:
    model = TwoTowerModel()
    score = model(user_vec, post_vec)          # (B, 1) scores
    weights = model.export_weights()           # dict for Go JSON checkpoint
"""
import torch
import torch.nn as nn
import torch.nn.functional as F


class Tower(nn.Module):
    def __init__(self, input_dim: int = 1536, hidden: int = 256, out_dim: int = 128):
        super().__init__()
        self.net = nn.Sequential(
            nn.Linear(input_dim, hidden),
            nn.ReLU(),
            nn.Linear(hidden, out_dim),
        )

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        return F.normalize(self.net(x), dim=-1)


class TwoTowerModel(nn.Module):
    def __init__(
        self,
        input_dim: int = 1536,
        hidden: int = 256,
        repr_dim: int = 128,
    ):
        super().__init__()
        self.input_dim = input_dim
        self.repr_dim = repr_dim
        self.user_tower = Tower(input_dim, hidden, repr_dim)
        self.post_tower = Tower(input_dim, hidden, repr_dim)

    def forward(
        self, user_vec: torch.Tensor, post_vec: torch.Tensor
    ) -> torch.Tensor:
        """
        Args:
            user_vec: (B, input_dim)
            post_vec: (B, input_dim)
        Returns:
            score: (B, 1) relevance in [0, 1]
        """
        user_repr = self.user_tower(user_vec)   # (B, repr_dim), unit norm
        post_repr = self.post_tower(post_vec)   # (B, repr_dim), unit norm
        dot = (user_repr * post_repr).sum(dim=-1, keepdim=True)  # (B, 1) in [-1,1]
        return (dot + 1.0) / 2.0               # map to [0, 1]

    def export_weights(self) -> dict:
        """
        Return the first linear layer's weights for Go JSON checkpoint.

        The Go Ranker uses a single-layer projection (no activation) followed
        by L2 normalisation — a lightweight approximation of the full tower.
        For higher fidelity, export via ONNX instead.
        """
        uw = self.user_tower.net[0].weight.detach().cpu().numpy()
        pw = self.post_tower.net[0].weight.detach().cpu().numpy()
        return {
            "input_dim": int(uw.shape[1]),
            "repr_dim": int(uw.shape[0]),
            "user_weights": uw.tolist(),
            "post_weights": pw.tolist(),
        }
