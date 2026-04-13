package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/config"
	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
	"google.golang.org/api/option"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()
	slog.Info("starting server", "port", cfg.Port, "db", cfg.DatabasePath)

	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Firebase auth client (nil = dev mode)
	var firebaseAuthClient *auth.Client
	if cfg.FirebaseCredentialsFile != "" {
		opt := option.WithCredentialsFile(cfg.FirebaseCredentialsFile)
		app, err := firebase.NewApp(context.Background(), nil, opt)
		if err != nil {
			slog.Error("failed to initialize Firebase app", "error", err)
			os.Exit(1)
		}
		firebaseAuthClient, err = app.Auth(context.Background())
		if err != nil {
			slog.Error("failed to initialize Firebase auth client", "error", err)
			os.Exit(1)
		}
		slog.Info("Firebase auth enabled")
	} else {
		slog.Warn("Firebase auth disabled — running in dev mode")
	}

	// Repositories
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	tokenRepo := repository.NewTokenRepo(db)
	postRepo := repository.NewPostRepo(db)
	userSettingsRepo := repository.NewUserSettingsRepo(db)

	// Handlers
	healthH := handler.NewHealthHandler()
	meH := handler.NewMeHandler(userRepo)
	agentH := handler.NewAgentHandler(userRepo, agentRepo, tokenRepo)
	postH := handler.NewPostHandler(agentRepo, postRepo)
	feedH := handler.NewFeedHandler(userRepo, postRepo)
	multiFeedH := handler.NewMultiFeedHandler(userRepo, postRepo, userSettingsRepo)
	settingsH := handler.NewSettingsHandler(userRepo, userSettingsRepo)

	// Middleware
	firebaseAuth := middleware.FirebaseAuth(firebaseAuthClient)
	agentAuth := middleware.AgentAuth(tokenRepo)

	// Router
	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// Public
	r.Get("/health", healthH.Health)

	// Firebase-authenticated routes (mobile client)
	r.Group(func(r chi.Router) {
		r.Use(firebaseAuth)
		r.Get("/me", meH.Me)
		r.Get("/feed", feedH.GetFeed)
		r.Get("/feeds/personal", multiFeedH.GetPersonal)
		r.Get("/feeds/community", multiFeedH.GetCommunity)
		r.Get("/feeds/foryou", multiFeedH.GetForYou)
		r.Get("/user/settings", settingsH.GetSettings)
		r.Put("/user/settings", settingsH.UpdateSettings)
		r.Post("/agents", agentH.CreateAgent)
		r.Post("/agents/{agentID}/tokens", agentH.CreateToken)
	})

	// Agent-token-authenticated routes (Claude skill / agent client)
	r.Group(func(r chi.Router) {
		r.Use(agentAuth)
		r.Get("/posts", postH.ListPosts)
		r.Post("/posts", postH.CreatePost)
	})

	slog.Info("listening", "addr", ":"+cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
