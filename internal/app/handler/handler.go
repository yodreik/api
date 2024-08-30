package handler

import (
	"api/internal/config"
	"api/internal/repository"
	"api/internal/token"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	config     *config.Config
	repository *repository.Repository
	token      *token.Manager
}

func New(c *config.Config, r *repository.Repository) *Handler {
	return &Handler{
		config:     c,
		repository: r,
		token:      token.New(c.Token.Secret),
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
