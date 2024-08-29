package handler

import (
	"api/internal/app/handler/response"
	"api/internal/app/handler/response/responsebody"
	"api/internal/lib/sl"
	repoerr "api/internal/repository/errors"
	"api/pkg/requestid"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// @Summary      Get information about current user
// @Description  returns an user's information, that currently logged in
// @Security     AccessToken
// @Tags         auth
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

	authHeader := ctx.GetHeader("Authorization")
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 {
		log.Info("Incorrect authorization header", slog.String("authorization", authHeader))
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.Err("invalid authorization header"))
		return
	}

	if parts[0] != "Bearer" {
		log.Info("Incorrect type of authorization token", slog.String("type", parts[0]))
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.Err("invalid authorization token type"))
		return
	}

	accessToken := parts[1]

	token, err := jwt.Parse(accessToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(h.config.Token.Secret), nil
	})
	if err != nil {
		log.Error("Can't parse JWT token", slog.String("token", accessToken), sl.Err(err))
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.Err("invalid authorization token"))
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		log.Error("Can't parse JWT token", slog.String("token", accessToken))
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.Err("invalid authorization token"))
		return
	}
	userID := claims["id"].(string)

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
