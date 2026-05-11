package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"go.uber.org/zap"

	"github.com/eguilde/egueducation/internal/admin"
	"github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/config"
	"github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/eguilde/egueducation/internal/notification"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync() //nolint:errcheck

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("database connection failed", zap.Error(err))
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		logger.Fatal("database migration failed", zap.Error(err))
	}

	smsService := notification.NewSMSService(cfg.SMSAPIToken, cfg.SMSSenderName)
	authService := auth.NewService(cfg, smsService, pool)
	adminService := admin.NewService(cfg)

	router := chi.NewRouter()
	router.Use(chimw.RequestID)
	router.Use(chimw.RealIP)
	router.Use(chimw.Recoverer)
	router.Use(chimw.Compress(5))
	router.Use(httprate.LimitByIP(120, time.Minute))
	router.Use(cors(cfg.FrontendOrigin))

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		dbStatus := "ok"
		if err := pool.Ping(r.Context()); err != nil {
			dbStatus = "error"
		}
		httpx.JSON(w, http.StatusOK, map[string]any{
			"status":   "ok",
			"service":  "egueducation-api",
			"database": dbStatus,
			"time":     time.Now().UTC(),
		})
	})

	router.Route("/api", func(r chi.Router) {
		r.Get("/meta/app", func(w http.ResponseWriter, r *http.Request) {
			httpx.JSON(w, http.StatusOK, map[string]any{
				"name":              "EguEducation",
				"default_locale":    "ro",
				"available_locales": []string{"ro", "en"},
				"theme": map[string]string{
					"family": "material3-expressive",
					"brand":  "red-rose",
				},
			})
		})

		r.Get("/auth/methods", authService.ListMethods)
		r.Get("/auth/ui-config", authService.UIConfig)
		r.Get("/me", authService.SessionContext)
		r.Get("/admin/dashboard", adminService.Dashboard)
		r.Get("/admin/users", adminService.ListUsers)
		r.Get("/admin/users/filters", adminService.UserFilters)
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("egueducation api listening", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server shutdown failed", zap.Error(err))
	}
}

func cors(frontendOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", frontendOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, DPoP, X-Request-ID, Accept-Language")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
