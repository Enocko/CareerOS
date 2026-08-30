package server

import (
	"net/http"
	"time"

	"github.com/careeros/api/internal/applications"
	"github.com/careeros/api/internal/auth"
	"github.com/careeros/api/internal/config"
	"github.com/careeros/api/internal/admin"
	"github.com/careeros/api/internal/middleware"
	"github.com/careeros/api/internal/observability"
	"github.com/careeros/api/internal/opportunities"
	"github.com/careeros/api/internal/opportunityreports"
	"github.com/careeros/api/internal/platform"
	"github.com/careeros/api/internal/profile"
	"github.com/careeros/api/internal/notifications"
	"github.com/careeros/api/internal/recommendation"
	"github.com/careeros/api/internal/researchverification"
	"github.com/careeros/api/internal/saved"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewRouter creates the HTTP router with all middleware and routes.
func NewRouter(cfg *config.Config, pool *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recovery)
	r.Use(middleware.RequestID)
	r.Use(observability.HTTPMetricsMiddleware)
	r.Use(middleware.Logging)
	r.Use(middleware.SecurityHeaders(cfg.IsProduction()))
	r.Use(middleware.CORS(cfg.CORSOrigin))
	r.Use(middleware.CSRFOriginCheck(cfg.CORSOrigin))
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.NoCache)

	r.Get("/health", platform.LivenessHandler())
	r.Get("/ready", platform.ReadinessHandler(pool))
	if cfg.MetricsEnabled {
		r.With(middleware.MetricsAuth(cfg.MetricsToken)).Handle("/metrics", observability.MetricsHandler(pool))
	}

	profileRepo := profile.NewRepository(pool)
	profileService := profile.NewService(profileRepo)
	profileHandler := profile.NewHandler(profileService)

	authRepo := auth.NewRepository(pool)
	tokens := auth.NewTokenManager(cfg)
	authService := auth.NewService(authRepo, tokens, profileService)
	cookieCfg := auth.DefaultCookieConfig(cfg.CookieSecure, cfg.CookieSameSite)
	tokenExpiry := time.Duration(cfg.JWTExpiryHours) * time.Hour
	authHandler := auth.NewHandler(authService, cookieCfg, tokenExpiry)
	authLimiter := middleware.NewRateLimiter(20, time.Minute)
	reportLimiter := middleware.NewRateLimiter(10, time.Minute)

	oppRepo := opportunities.NewRepository(pool)
	oppService := opportunities.NewService(oppRepo)
	oppHandler := opportunities.NewHandler(oppService)

	recRepo := recommendation.NewRepository(pool)
	recService := recommendation.NewService(recRepo, profileRepo)
	recHandler := recommendation.NewHandler(recService)

	notifRepo := notifications.NewRepository(pool)
	notifService := notifications.NewService(notifRepo)
	notifHandler := notifications.NewHandler(notifService)

	savedRepo := saved.NewRepository(pool)
	savedService := saved.NewService(savedRepo)
	savedHandler := saved.NewHandler(savedService)

	appRepo := applications.NewRepository(pool)
	appService := applications.NewService(appRepo)
	appHandler := applications.NewHandler(appService)

	researchVerifRepo := researchverification.NewRepository(pool)
	researchVerifService := researchverification.NewService(researchVerifRepo)
	researchVerifHandler := researchverification.NewHandler(researchVerifService)

	reportRepo := opportunityreports.NewRepository(pool)
	reportService := opportunityreports.NewService(reportRepo)
	reportHandler := opportunityreports.NewHandler(reportService)
	adminHandler := admin.NewHandler(pool, reportService, researchVerifService)

	adminGuard := middleware.RequireAdmin(cfg.AdminEmailSet())
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.With(authLimiter.LimitByIP).Post("/register", authHandler.Register)
			r.With(authLimiter.LimitByIP).Post("/login", authHandler.Login)
			r.With(middleware.Authenticate(tokens)).Post("/logout", authHandler.Logout)
			r.With(middleware.Authenticate(tokens)).Get("/me", authHandler.Me)
		})

		r.Route("/profile", func(r chi.Router) {
			r.Use(middleware.Authenticate(tokens))
			r.Get("/", profileHandler.Get)
			r.Put("/", profileHandler.Update)
		})

		r.Route("/opportunities", func(r chi.Router) {
			r.Use(middleware.Authenticate(tokens))
			r.Get("/recommended", recHandler.List)
			r.Post("/recommended/events", recHandler.RecordEvent)
			r.Get("/", oppHandler.List)
			r.Get("/{id}", oppHandler.GetByID)
			r.Post("/{id}/save", savedHandler.Save)
			r.Delete("/{id}/save", savedHandler.Unsave)
			r.With(reportLimiter.LimitByIP).Post("/{id}/report", reportHandler.Create)
		})

		r.With(middleware.Authenticate(tokens)).Get("/saved-opportunities", savedHandler.List)

		r.Route("/applications", func(r chi.Router) {
			r.Use(middleware.Authenticate(tokens))
			r.Post("/", appHandler.Create)
			r.Get("/", appHandler.List)
			r.Get("/{id}", appHandler.GetByID)
			r.Patch("/{id}", appHandler.Update)
			r.Delete("/{id}", appHandler.Delete)
		})

		r.Route("/notifications", func(r chi.Router) {
			r.Use(middleware.Authenticate(tokens))
			r.Get("/", notifHandler.List)
			r.Get("/unread-count", notifHandler.UnreadCount)
			r.Post("/mark-all-read", notifHandler.MarkAllRead)
			r.Patch("/{id}/read", notifHandler.MarkRead)
		})

		r.Route("/admin/research", func(r chi.Router) {
			r.Use(middleware.Authenticate(tokens))
			r.Use(adminGuard)
			r.Get("/queue", researchVerifHandler.ListQueue)
			r.Get("/metrics", researchVerifHandler.GetMetrics)
			r.Get("/opportunities/{id}", researchVerifHandler.GetOpportunity)
			r.Get("/opportunities/{id}/verifications", researchVerifHandler.ListVerifications)
			r.Post("/opportunities/{id}/verify", researchVerifHandler.Verify)
		})

		r.Route("/admin", func(r chi.Router) {
			r.Use(middleware.Authenticate(tokens))
			r.Use(adminGuard)
			r.Get("/overview", adminHandler.GetOverview)
			r.Get("/reports", reportHandler.ListQueue)
			r.Patch("/reports/opportunities/{id}", reportHandler.ResolveOpportunity)
		})
	})

	return r
}
