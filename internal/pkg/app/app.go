package app

import (
	"api/internal/app/router"
	"api/internal/config"
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
	gin.SetMode(gin.ReleaseMode) // Turn off gin's logs

	slog.Info("Server running")

	r := router.New(a.config)

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
				slog.Error("Failed to start server", slog.String("error", err.Error()))
			}
		}
	}()

	slog.Info("Server started", slog.String("address", server.Addr))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	slog.Info("Server shutting down")

	err := server.Shutdown(context.Background())
	if err != nil {
		slog.Error("Error occurred on server shutting down", slog.String("error", err.Error()))
	}

	slog.Info("Server stopped")
}
