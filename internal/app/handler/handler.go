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
	repository *repository.Repository
	mailer     *mailer.Mailer
	token      *token.Manager
}

func New(c *config.Config, r *repository.Repository) *Handler {
	return &Handler{
		repository: r,
		mailer:     mailer.New(c.Mail),
		token:      token.New(c.Token.Secret),
	}
}

// @Summary      Ping a server
// @Description  check if server status is ok
// @Tags         status
// @Accept       json
// @Produce      json
// @Success      200 {string}    string "ok"
// @Router       /healthcheck    [get]
func (h *Handler) Healthcheck(ctx *gin.Context) {
	ctx.String(http.StatusOK, "ok")
}
