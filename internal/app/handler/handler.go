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

// @Summary      Ping a server
// @Description  check if server status is ok
// @Accept       json
// @Produce      json
// @Success      200 {string}    string "ok"
// @Router       /healthcheck    [get]
func (h *Handler) Healthcheck(ctx *gin.Context) {
	ctx.String(http.StatusOK, "ok")
}
