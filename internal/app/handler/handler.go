package handler

import (
	"api/internal/config"
	"api/internal/repository"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	config     *config.Config
	repository *repository.Repository
}

func New(c *config.Config, r *repository.Repository) *Handler {
	return &Handler{
		config:     c,
		repository: r,
	}
}

func (h *Handler) Healthcheck(ctx *gin.Context) {
	// TODO: Create a middleware for logging requests
	slog.Info("Request to /healthcheck handled")

	ctx.String(http.StatusOK, "OK")
}
