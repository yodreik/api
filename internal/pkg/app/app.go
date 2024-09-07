package app

import (
	"api/internal/app/router"
	"api/internal/config"
	"api/internal/lib/logger/prettyslog"
	"api/internal/lib/logger/sl"
	"api/internal/mailer"
	"api/internal/repository"
	"api/internal/repository/postgres"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
)

type App struct {
	config *config.Config
}

func New(c *config.Config) *App {
	return &App{
		config: c,
	}
}

func (a *App) Run() {
	var logger *slog.Logger
	switch a.config.Env {
	case config.EnvLocal:
		logger = prettyslog.Init()
	case config.EnvDevelopment:
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	case config.EnvProduction:
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}

	slog.SetDefault(logger)

	gin.SetMode(gin.ReleaseMode) // Turn off gin's logs

	slog.Info("Starting API server...", slog.String("env", a.config.Env))

	db, err := postgres.New(&a.config.Postgres)
	if err != nil {
		slog.Error("Could not connect to PostgreSQL", sl.Err(err))
		os.Exit(1)
	}

	slog.Info("Successfully connected to PostgreSQL")

	repo := repository.New(db)

	m := mailer.New(a.config.Mail)

	r := router.New(a.config, repo, m)

	server := &http.Server{
		Addr:         a.config.Server.Address,
		Handler:      r.InitRoutes(),
		ReadTimeout:  a.config.Server.Timeout,
		WriteTimeout: a.config.Server.Timeout,
		IdleTimeout:  a.config.Server.IdleTimeout,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				slog.Error("Failed to start server", sl.Err(err))
				os.Exit(1)
			}
		}
	}()

	slog.Info("Server started", slog.String("address", server.Addr))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	slog.Info("Server shutting down")

	err = server.Shutdown(context.Background())
	if err != nil {
		slog.Error("Error occurred on server shutting down", sl.Err(err))
		os.Exit(1)
	}

	slog.Info("API server stopped")

	err = db.Close()
	if err != nil {
		slog.Error("Could not close PostgreSQL connection properly", sl.Err(err))
		os.Exit(1)
	}

	slog.Info("PostgreSQL connection closed")
}
