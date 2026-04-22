package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/config"
	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/embedding"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg := config.Load()

	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		slog.Error("open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	repo := embedding.NewEmbeddingRepo(db)
	embedder := embedding.NewEmbedderFromConfig(embedding.ProviderConfig{
		Provider:             cfg.EmbeddingProvider,
		FallbackProvider:     cfg.EmbeddingFallbackProvider,
		GoogleAPIKey:         cfg.GoogleAPIKey,
		Model:                cfg.EmbeddingModel,
		OutputDimensionality: cfg.EmbeddingOutputDimensionality,
		AllowImageURLParts:   cfg.EmbeddingAllowImageURLParts,
	})

	worker := embedding.NewBackfillWorker(repo, embedder, 100)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := worker.Run(ctx); err != nil {
		slog.Error("embedding backfill failed", "error", err)
		os.Exit(1)
	}
	slog.Info("embedding backfill succeeded")
}
