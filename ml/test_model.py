"""
TDD tests for TwoTowerModel.

Run with: pytest ml/test_model.py -v

These tests define the required behaviour. All fail until model.py is implemented.
"""
import json
import os
import tempfile

import pytest
import torch


# ---------------------------------------------------------------------------
# fixtures
# ---------------------------------------------------------------------------

@pytest.fixture
def model():
    from model import TwoTowerModel
    return TwoTowerModel()


@pytest.fixture
def trained_model():
    """Return a model trained on a tiny synthetic dataset (200 pairs)."""
    from model import TwoTowerModel
    from train import train_synthetic
    m = TwoTowerModel(input_dim=64, hidden=32, repr_dim=16)
    train_synthetic(m, n_pairs=200, epochs=30, lr=1e-2)
    return m


# ---------------------------------------------------------------------------
# architecture tests
# ---------------------------------------------------------------------------

def test_model_input_output_shapes(model):
    user_vec = torch.randn(1, 1536)
    post_vec = torch.randn(1, 1536)
    score = model(user_vec, post_vec)
    assert score.shape == (1, 1), f"expected (1,1), got {score.shape}"


def test_score_range_unit_bounded(model):
    """Score must always be in [0, 1] — we map dot-product via (x+1)/2."""
    user_vecs = torch.randn(64, 1536)
    post_vecs = torch.randn(64, 1536)
    scores = model(user_vecs, post_vecs)
    assert (scores >= 0).all() and (scores <= 1).all(), \
        f"scores out of [0,1]: min={scores.min():.4f} max={scores.max():.4f}"


def test_user_tower_output_is_unit_vector(model):
    x = torch.randn(4, 1536)
    repr_ = model.user_tower(x)
    norms = torch.norm(repr_, dim=-1)
    assert torch.allclose(norms, torch.ones(4), atol=1e-5), \
        f"user tower not L2-normalised: {norms}"


def test_post_tower_output_is_unit_vector(model):
    x = torch.randn(4, 1536)
    repr_ = model.post_tower(x)
    norms = torch.norm(repr_, dim=-1)
    assert torch.allclose(norms, torch.ones(4), atol=1e-5), \
        f"post tower not L2-normalised: {norms}"


def test_batch_inference_correct_count(model):
    """ScoreBatch must return one score per pair, not a cross-product matrix."""
    user_vecs = torch.randn(32, 1536)
    post_vecs = torch.randn(32, 1536)
    scores = model(user_vecs, post_vecs)
    assert scores.shape == (32, 1), f"expected (32,1), got {scores.shape}"


# ---------------------------------------------------------------------------
# post-training behaviour
# ---------------------------------------------------------------------------

def test_score_range_after_training(trained_model):
    """After brief training on synthetic data all scores must stay in [0,1]."""
    user_vecs = torch.randn(50, 64)
    post_vecs = torch.randn(50, 64)
    with torch.no_grad():
        scores = trained_model(user_vecs, post_vecs)
    assert (scores >= 0).all() and (scores <= 1).all()


def test_positive_pairs_score_higher_than_negatives(trained_model):
    """Mean positive score > mean negative score after training."""
    torch.manual_seed(42)
    n = 100
    user_vecs = torch.randn(n, 64)
    # positives: same direction
    pos_vecs = user_vecs + 0.05 * torch.randn(n, 64)
    # negatives: random direction
    neg_vecs = torch.randn(n, 64)

    with torch.no_grad():
        pos_scores = trained_model(user_vecs, pos_vecs)
        neg_scores = trained_model(user_vecs, neg_vecs)

    assert pos_scores.mean() > neg_scores.mean(), (
        f"pos mean {pos_scores.mean():.3f} not > neg mean {neg_scores.mean():.3f}"
    )


# ---------------------------------------------------------------------------
# export / serialisation
# ---------------------------------------------------------------------------

def test_json_export_valid_structure(model):
    """export_weights() must return a dict with the required keys."""
    weights = model.export_weights()
    assert "input_dim" in weights
    assert "repr_dim" in weights
    assert "user_weights" in weights
    assert "post_weights" in weights
    assert len(weights["user_weights"]) == weights["repr_dim"]
    assert len(weights["user_weights"][0]) == weights["input_dim"]


def test_json_export_roundtrip(model):
    """Weights saved to JSON and reloaded must produce identical output."""
    from export import export_json, load_json_ranker

    with tempfile.TemporaryDirectory() as tmp:
        path = os.path.join(tmp, "ranker.json")
        export_json(model, path)

        loaded = load_json_ranker(path)

        vec = torch.randn(1, 1536)
        orig_repr = model.user_tower(vec).detach()
        loaded_repr = loaded.user_tower(vec).detach()
        assert torch.allclose(orig_repr, loaded_repr, atol=1e-5), \
            "reloaded model user tower output differs from original"


def test_model_file_size_under_limit(model):
    """Exported JSON checkpoint must be < 5 MB."""
    with tempfile.TemporaryDirectory() as tmp:
        path = os.path.join(tmp, "ranker.json")
        from export import export_json
        export_json(model, path)
        size = os.path.getsize(path)
        limit = 5 * 1024 * 1024
        assert size < limit, f"checkpoint {size} bytes >= 5MB limit"


# ---------------------------------------------------------------------------
# training quality
# ---------------------------------------------------------------------------

def test_model_trains_above_auc_threshold():
    """Brief training on 1 000 synthetic pairs must achieve val AUC > 0.70."""
    from model import TwoTowerModel
    from train import train_synthetic, evaluate_auc
    torch.manual_seed(0)
    m = TwoTowerModel(input_dim=64, hidden=32, repr_dim=16)
    train_synthetic(m, n_pairs=1000, epochs=50, lr=1e-2)
    auc = evaluate_auc(m, n_pairs=200, input_dim=64)
    assert auc > 0.70, f"val AUC {auc:.3f} < 0.70 threshold"
