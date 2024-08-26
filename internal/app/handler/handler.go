package handler

import (
	"api/internal/config"
	"api/internal/repository"
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
	ctx.String(http.StatusOK, "OK")
}
