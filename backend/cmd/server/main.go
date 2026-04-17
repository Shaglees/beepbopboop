package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/config"
	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
	"github.com/shanegleeson/beepbopboop/backend/internal/weather"
	"google.golang.org/api/option"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()
	slog.Info("starting server", "port", cfg.Port, "db", cfg.DatabaseURL)

	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}

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
	eventRepo := repository.NewEventRepo(db)
	weightsRepo := repository.NewWeightsRepo(db)
	templateRepo := repository.NewTemplateRepo(db)
	reactionRepo := repository.NewReactionRepo(db)

	// Handlers
	healthH := handler.NewHealthHandler()
	meH := handler.NewMeHandler(userRepo)
	agentH := handler.NewAgentHandler(userRepo, agentRepo, tokenRepo)
	postH := handler.NewPostHandler(agentRepo, postRepo)
	feedH := handler.NewFeedHandler(userRepo, postRepo)
	multiFeedH := handler.NewMultiFeedHandler(userRepo, postRepo, userSettingsRepo, weightsRepo, eventRepo, reactionRepo)
	settingsH := handler.NewSettingsHandler(userRepo, userSettingsRepo)
	eventsH := handler.NewEventsHandler(userRepo, agentRepo, eventRepo)
	weightsH := handler.NewWeightsHandler(agentRepo, weightsRepo)
	templatesH := handler.NewTemplatesHandler(userRepo, agentRepo, templateRepo)
	reactionsH := handler.NewReactionsHandler(userRepo, agentRepo, reactionRepo)
	weatherSvc := weather.NewService()

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
		r.Post("/posts/{postID}/events", eventsH.TrackEvent)
		r.Post("/events/batch", eventsH.BatchTrack)
		r.Put("/posts/{postID}/reaction", reactionsH.SetReaction)
		r.Delete("/posts/{postID}/reaction", reactionsH.RemoveReaction)
		r.Get("/user/templates", templatesH.ListTemplatesFirebase)
	})

	// Agent-token-authenticated routes (Claude skill / agent client)
	r.Group(func(r chi.Router) {
		r.Use(agentAuth)
		r.Get("/posts", postH.ListPosts)
		r.Get("/posts/stats", postH.GetPostStats)
		r.Post("/posts", postH.CreatePost)
		r.Post("/posts/lint", postH.LintPost)
		r.Get("/events/summary", eventsH.Summary)
		r.Get("/reactions/summary", reactionsH.Summary)
		r.Get("/user/weights", weightsH.GetWeights)
		r.Put("/user/weights", weightsH.UpdateWeights)
		r.Get("/user/templates", templatesH.ListTemplatesAgent)
		r.Put("/user/templates/{hint}", templatesH.UpsertTemplate)
		r.Delete("/user/templates/{hint}", templatesH.DeleteTemplate)
	})

	// Background weather worker — fetches weather for active user locations
	// and upserts posts so they appear in nearby users' feeds.
	workerCtx, workerCancel := context.WithCancel(context.Background())
	weatherWorker := weather.NewWorker(weatherSvc, postRepo, userSettingsRepo, 30*time.Minute)
	go weatherWorker.Run(workerCtx)

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}

	go func() {
		slog.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("shutting down", "signal", sig.String())

	workerCancel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	db.Close()
	slog.Info("server stopped")
}
