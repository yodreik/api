package handler

import (
	"api/internal/app/handler/response"
	"api/internal/app/handler/response/responsebody"
	"api/internal/lib/logger/sl"
	repoerr "api/internal/repository/errors"
	"api/pkg/requestid"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// @Summary      Get information about current user
// @Description  returns an user's information, that currently logged in
// @Security     AccessToken
// @Tags         user
// @Produce      json
// @Success      200 {object}   responsebody.User
// @Failure      401 {object}   responsebody.Message
// @Router       /auth/account  [get]
func (h *Handler) GetCurrentAccount(c *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.GetCurrentAccount"),
		slog.String("request_id", requestid.Get(c)),
	)

	userID := c.GetString("UserID")
	user, err := h.repository.User.GetByID(c, userID)
	if errors.Is(err, repoerr.ErrUserNotFound) {
		log.Debug("user not found", slog.String("id", userID))
		response.WithMessage(c, http.StatusUnauthorized, "invalid authorization token")
		return
	}
	if err != nil {
		log.Error("can't find user", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	resUser := responsebody.User{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
	}

	c.JSON(http.StatusOK, resUser)
}
