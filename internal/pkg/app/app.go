package app

import (
	"api/internal/app/router"
	"api/internal/config"
	"api/internal/lib/logger/prettyslog"
	"api/internal/lib/logger/sl"
	"api/internal/mailer"
	"api/internal/repository"
	"api/internal/repository/postgres"
	"api/internal/token"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	ctx := context.Background()

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

	slog.Info("starting API server...", slog.String("env", a.config.Env))

	db, err := postgres.New(&a.config.Postgres)
	if err != nil {
		slog.Error("could not connect to PostgreSQL", sl.Err(err))
		os.Exit(1)
	}

	slog.Info("successfully connected to PostgreSQL")

	repo := repository.New(db)
	m := mailer.New(a.config)
	tokenManager := token.New(a.config.Token)

	r := router.New(a.config, repo, m, tokenManager)

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
				slog.Error("failed to start server", sl.Err(err))
				os.Exit(1)
			}
		}
	}()

	slog.Info("server started", slog.String("address", server.Addr))

	go func() {
		for {
			n, err := repo.User.RemoveExpiredRecords(ctx)
			if err != nil {
				slog.Error("failed to delete expired records", sl.Err(err))
			} else if n > 0 {
				slog.Debug("expired records deleted", slog.Int64("count", n))
			}
			time.Sleep(1 * time.Hour)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	slog.Info("server shutting down")

	err = server.Shutdown(ctx)
	if err != nil {
		slog.Error("error occurred on server shutting down", sl.Err(err))
		os.Exit(1)
	}

	slog.Info("API server stopped")

	err = db.Close()
	if err != nil {
		slog.Error("could not close PostgreSQL connection properly", sl.Err(err))
		os.Exit(1)
	}

	slog.Info("connection to PostgreSQL closed")
}
