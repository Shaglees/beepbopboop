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
	"github.com/shanegleeson/beepbopboop/backend/internal/ab"
	"github.com/shanegleeson/beepbopboop/backend/internal/calendar"
	"github.com/shanegleeson/beepbopboop/backend/internal/config"
	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/embedding"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/ranking"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
	"github.com/shanegleeson/beepbopboop/backend/internal/scheduler"
	"github.com/shanegleeson/beepbopboop/backend/internal/sports"
	"github.com/shanegleeson/beepbopboop/backend/internal/videohealth"
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
	embeddingRepo := embedding.NewEmbeddingRepo(db)
	userSettingsRepo := repository.NewUserSettingsRepo(db)
	eventRepo := repository.NewEventRepo(db)
	weightsRepo := repository.NewWeightsRepo(db)
	templateRepo := repository.NewTemplateRepo(db)
	reactionRepo := repository.NewReactionRepo(db)
	pushTokenRepo := repository.NewPushTokenRepo(db)
	modelVersionRepo := ranking.NewModelVersionRepo(db)
	feedbackRepo := repository.NewFeedbackRepo(db)
	calendarRepo := repository.NewCalendarRepo(db)
	followRepo := repository.NewFollowRepo(db)
	videoRepo := repository.NewVideoRepo(db)
	userEmbeddingRepo := repository.NewUserEmbeddingRepo(db)
	postEmbeddingRepo := repository.NewPostEmbeddingRepo(db)

	var ranker *ranking.Ranker
	if cfg.RankerModelPath != "" {
		var err error
		ranker, err = ranking.NewRanker(cfg.RankerModelPath)
		if err != nil {
			slog.Warn("RANKER_MODEL_PATH set but ranker failed to load; rule-only ForYou",
				"path", cfg.RankerModelPath, "error", err)
			ranker = nil
		}
		if ranker != nil {
			postRepo.SetML(&repository.PostRepoML{
				Ranker:  ranker,
				PostEmb: postEmbeddingRepo,
				Blend:   cfg.MLRankBlend,
			})
			slog.Info("ForYou ML ranking enabled",
				"path", cfg.RankerModelPath, "input_dim", ranker.InputDim(), "blend", cfg.MLRankBlend)
		}
	}

	userEmbFront := repository.NewEmbeddingCacheFromLoader(userEmbeddingRepo, 1000, 5*time.Minute)

	// Handlers
	healthH := handler.NewHealthHandler()
	meH := handler.NewMeHandler(userRepo)
	agentH := handler.NewAgentHandler(userRepo, agentRepo, tokenRepo)
	postH := handler.NewPostHandler(agentRepo, postRepo, videoRepo)

	embedder := embedding.NewEmbedderFromConfig(embedding.ProviderConfig{
		Provider:             cfg.EmbeddingProvider,
		FallbackProvider:     cfg.EmbeddingFallbackProvider,
		GoogleAPIKey:         cfg.GoogleAPIKey,
		Model:                cfg.EmbeddingModel,
		OutputDimensionality: cfg.EmbeddingOutputDimensionality,
		AllowImageURLParts:   cfg.EmbeddingAllowImageURLParts,
	})
	postH.SetEmbeddingPipeline(embeddingRepo, embedder)
	if mv, ok := embedder.(embedding.ModelVersioner); ok {
		slog.Info("post embedding pipeline enabled", "provider", cfg.EmbeddingProvider, "model_version", mv.ModelVersion())
	} else {
		slog.Info("post embedding pipeline enabled", "provider", cfg.EmbeddingProvider)
	}

	feedH := handler.NewFeedHandler(userRepo, postRepo)
	multiFeedH := handler.NewMultiFeedHandler(userRepo, postRepo, userSettingsRepo, weightsRepo, eventRepo, reactionRepo, followRepo, userEmbFront)
	followH := handler.NewFollowHandler(userRepo, followRepo)
	settingsH := handler.NewSettingsHandler(userRepo, userSettingsRepo)
	eventsH := handler.NewEventsHandler(userRepo, agentRepo, eventRepo)
	weightsH := handler.NewWeightsHandler(agentRepo, userRepo, weightsRepo)
	weightsSummaryH := handler.NewWeightsSummaryHandler(userRepo, weightsRepo, eventRepo)
	templatesH := handler.NewTemplatesHandler(userRepo, agentRepo, templateRepo)
	reactionsH := handler.NewReactionsHandler(userRepo, agentRepo, reactionRepo)
	pushTokenH := handler.NewPushTokenHandler(userRepo, pushTokenRepo)
	calendarH := handler.NewCalendarHandler(userRepo, calendarRepo, userSettingsRepo)
	deploymentGate := ranking.NewDeploymentGate(0.02)
	mlAdminH := handler.NewMLAdminHandler(modelVersionRepo, deploymentGate, os.Getenv("OPERATOR_AGENT_ID"))
	weatherSvc := weather.NewService()
	sportsSvc := sports.NewService()
	sportsH := handler.NewSportsHandler(sportsSvc)
	feedbackH := handler.NewFeedbackHandler(userRepo, feedbackRepo)
	userEmbedder := embedding.NewUserEmbedder(db, userEmbeddingRepo)

	creatorRepo := repository.NewLocalCreatorRepo(db)
	creatorsH := handler.NewCreatorsHandler(creatorRepo, userRepo, userSettingsRepo)

	experimentRepo := repository.NewExperimentRepo(db)
	abAssigner := ab.NewAssigner(db)
	experimentsH := handler.NewExperimentsHandler(abAssigner, userRepo, experimentRepo)

	prototypeStore := embedding.NewPrototypeStore(db)
	go func() {
		if err := prototypeStore.Compute(context.Background()); err != nil {
			slog.Warn("prototype store: initial compute failed", "error", err)
		}
	}()
	onboardingH := handler.NewOnboardingHandler(userRepo, prototypeStore, userEmbeddingRepo)

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
		r.Get("/feeds/following", multiFeedH.GetFollowing)
		r.Get("/posts/saved", multiFeedH.GetSaved)
		r.Get("/user/settings", settingsH.GetSettings)
		r.Put("/user/settings", settingsH.UpdateSettings)
		r.Get("/user/weights", weightsH.GetWeightsFirebase)
		r.Put("/user/weights", weightsH.UpdateWeightsFirebase)
		r.Put("/user/push-token", pushTokenH.RegisterPushToken)
		r.Post("/user/calendar-events", calendarH.SyncCalendarEvents)
		r.Get("/user/digest", pushTokenH.GetDigestPosts)
		r.Post("/agents", agentH.CreateAgent)
		r.Post("/agents/{agentID}/tokens", agentH.CreateToken)
		r.Get("/agents/following", followH.ListFollowing)
		r.Post("/agents/{agentID}/follow", followH.Follow)
		r.Delete("/agents/{agentID}/follow", followH.Unfollow)
		r.Get("/agents/{agentID}", followH.GetAgentProfile)
		r.Post("/posts/{postID}/events", eventsH.TrackEvent)
		r.Post("/events/batch", eventsH.BatchTrack)
		r.Put("/posts/{postID}/reaction", reactionsH.SetReaction)
		r.Delete("/posts/{postID}/reaction", reactionsH.RemoveReaction)
		r.Get("/user/templates", templatesH.ListTemplatesFirebase)
		r.Get("/user/weights/summary", weightsSummaryH.GetSummary)
		r.Get("/sports/scores", sportsH.GetScores)
		r.Post("/posts/{postID}/response", feedbackH.SubmitResponse)
		r.Get("/posts/{postID}/responses", feedbackH.GetResponses)
		r.Get("/creators/nearby", creatorsH.GetNearby)
		r.Post("/user/interests", onboardingH.SubmitInterests)
		r.Get("/experiments/{name}/variant", experimentsH.GetVariant)
	})

	// Agent-token-authenticated routes (Claude skill / agent client)
	r.Group(func(r chi.Router) {
		r.Use(agentAuth)
		r.Get("/posts", postH.ListPosts)
		r.Get("/posts/stats", postH.GetPostStats)
		r.Post("/posts", postH.CreatePost)
		r.Post("/posts/lint", postH.LintPost)
		r.Post("/creators", creatorsH.Create)
		r.Get("/events/summary", eventsH.Summary)
		r.Get("/reactions/summary", reactionsH.Summary)
		r.Get("/user/weights", weightsH.GetWeights)
		r.Put("/user/weights", weightsH.UpdateWeights)
		r.Get("/user/templates", templatesH.ListTemplatesAgent)
		r.Put("/user/templates/{hint}", templatesH.UpsertTemplate)
		r.Delete("/user/templates/{hint}", templatesH.DeleteTemplate)
		r.Post("/admin/experiments", experimentsH.CreateExperiment)
		r.Get("/admin/experiments/{name}/results", experimentsH.GetResults)
		r.Get("/admin/ml/versions", mlAdminH.ListVersions)
		r.Post("/admin/ml/models/{id}/deploy", mlAdminH.DeployVersion)
	})

	workerCtx, workerCancel := context.WithCancel(context.Background())
	retrainWorker := ranking.NewRetrainWorker(modelVersionRepo, 1000, 7*24*time.Hour)
	go retrainWorker.Run(workerCtx)
	weatherWorker := weather.NewWorker(weatherSvc, postRepo, userSettingsRepo, 30*time.Minute)
	go weatherWorker.Run(workerCtx)

	sportsWorker := sports.NewWorker(sportsSvc, postRepo, 10*time.Minute)
	go sportsWorker.Run(workerCtx)

	schedulerWorker := scheduler.NewWorker(postRepo, 1*time.Minute)
	go schedulerWorker.Run(workerCtx)

	calendarWorker := calendar.NewWorker(calendarRepo, postRepo, userSettingsRepo, 6*time.Hour)
	go calendarWorker.Run(workerCtx)

	embeddingWorker := embedding.NewWorker(userEmbedder, 24*time.Hour)
	go embeddingWorker.Run(workerCtx)

	videoHealthWorker := videohealth.NewScheduledWorker(videoRepo, videohealth.NewHTTPChecker(nil), 6*time.Hour)
	go videoHealthWorker.Run(workerCtx)

	if ranker != nil {
		ranker.StartWatcher(workerCtx, cfg.RankerModelPath, 5*time.Minute)
	}

	// A/B guardrail: periodically check running experiments and pause on regression.
	abGuardrail := ab.NewGuardrail(db, ab.GuardrailConfig{SaveRateDropPct: 5, SessionDropPct: 20})
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-workerCtx.Done():
				return
			case <-ticker.C:
				rows, err := db.QueryContext(workerCtx,
					"SELECT name FROM ab_experiments WHERE status = 'running'")
				if err != nil {
					slog.Warn("guardrail: failed to list experiments", "error", err)
					continue
				}
				var names []string
				for rows.Next() {
					var n string
					if err := rows.Scan(&n); err == nil {
						names = append(names, n)
					}
				}
				rows.Close()
				for _, name := range names {
					if _, err := abGuardrail.CheckAndPause(workerCtx, name); err != nil {
						slog.Warn("guardrail: check failed", "experiment", name, "error", err)
					}
				}
			}
		}
	}()

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
