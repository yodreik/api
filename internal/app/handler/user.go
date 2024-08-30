package handler

import (
	"api/internal/app/handler/response"
	"api/internal/app/handler/response/responsebody"
	"api/internal/lib/sl"
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
// @Success      200 {object}  responsebody.User
// @Failure      400 {object}  responsebody.Error
// @Failure      404 {object}  responsebody.Error
// @Router       /me           [get]
func (h *Handler) Me(ctx *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.Me"),
		slog.String("request_id", requestid.Get(ctx)),
	)

	userID := ctx.GetString("UserID")
	user, err := h.repository.User.GetByID(ctx, userID)
	if errors.Is(err, repoerr.ErrUserNotFound) {
		log.Info("User not found", slog.String("id", userID))
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.Err("invalid authorization token"))
		return
	}
	if err != nil {
		log.Error("Can't find user", sl.Err(err))
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, response.Err("can't get me"))
		return
	}

	resUser := responsebody.User{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
	}

	ctx.JSON(http.StatusOK, resUser)
}
