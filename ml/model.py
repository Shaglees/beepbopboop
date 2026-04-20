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
        self.hidden = hidden
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
        """Return both projection layers and biases for the Go JSON checkpoint."""
        uw1 = self.user_tower.net[0].weight.detach().cpu().numpy()
        ub1 = self.user_tower.net[0].bias.detach().cpu().numpy()
        uw2 = self.user_tower.net[2].weight.detach().cpu().numpy()
        ub2 = self.user_tower.net[2].bias.detach().cpu().numpy()
        pw1 = self.post_tower.net[0].weight.detach().cpu().numpy()
        pb1 = self.post_tower.net[0].bias.detach().cpu().numpy()
        pw2 = self.post_tower.net[2].weight.detach().cpu().numpy()
        pb2 = self.post_tower.net[2].bias.detach().cpu().numpy()
        return {
            "input_dim": int(uw1.shape[1]),
            "hidden_dim": int(uw1.shape[0]),
            "repr_dim": int(uw2.shape[0]),
            "user_weights_1": uw1.tolist(),
            "user_bias_1": ub1.tolist(),
            "user_weights_2": uw2.tolist(),
            "user_bias_2": ub2.tolist(),
            "post_weights_1": pw1.tolist(),
            "post_bias_1": pb1.tolist(),
            "post_weights_2": pw2.tolist(),
            "post_bias_2": pb2.tolist(),
        }
