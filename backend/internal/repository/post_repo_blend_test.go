package repository

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/ranking"
)

func TestBlendForYouMLScores_NoMLReturnsRuleScores(t *testing.T) {
	r := &PostRepo{}
	rule := []float64{1.5, 2.5, 3.0}
	candidates := []model.Post{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	got := r.blendForYouMLScores(candidates, rule, []float32{1, 2, 3})
	if len(got) != 3 {
		t.Fatalf("len %d", len(got))
	}
	for i := range rule {
		if got[i] != rule[i] {
			t.Fatalf("idx %d: got %v want %v", i, got[i], rule[i])
		}
	}
}

func TestBlendForYouMLScores_ZeroBlendSkipsML(t *testing.T) {
	ranker, err := ranking.NewRanker("../ranking/testdata/ranker.json")
	if err != nil {
		t.Fatal(err)
	}
	r := &PostRepo{ml: &PostRepoML{Ranker: ranker, Blend: 0}}
	rule := []float64{3, 4}
	got := r.blendForYouMLScores([]model.Post{{ID: "a"}, {ID: "b"}}, rule, []float32{1, 0, 0, 0})
	if got[0] != 3 || got[1] != 4 {
		t.Fatalf("got %v", got)
	}
}
