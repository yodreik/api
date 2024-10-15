package handler

import (
	"api/internal/config"
	"api/internal/mailer"
	"api/internal/repository"
	"api/internal/token"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	config     *config.Config
	repository *repository.Repository
	mailer     mailer.Mailer
	token      token.Manager
}

func New(c *config.Config, r *repository.Repository, m mailer.Mailer, t token.Manager) *Handler {
	return &Handler{
		config:     c,
		repository: r,
		mailer:     m,
		token:      t,
	}
}

// @Summary      Ping a server
// @Description  check if server status is ok
// @Tags         status
// @Accept       json
// @Produce      json
// @Success      200 {string}    string "ok"
// @Router       /healthcheck    [get]
func (h *Handler) Healthcheck(c *gin.Context) {
	c.String(http.StatusOK, "ok")
}
