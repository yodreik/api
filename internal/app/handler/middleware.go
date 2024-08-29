package handler

import (
	"api/internal/app/handler/response"
	"api/internal/lib/sl"
	"api/pkg/requestid"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func (h *Handler) UserIdentity(ctx *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.UserIdentity"),
		slog.String("request_ud", requestid.Get(ctx)),
	)

	header := ctx.GetHeader("Authorization")
	parts := strings.Split(header, " ")
	if len(parts) != 2 {
		log.Info("Incorrect authorization header", slog.String("authorization", header))
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.Err("empty authorization header"))
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
		log.Error("Can't parse access token", slog.String("token", accessToken), sl.Err(err))
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.Err("invalid authorization token"))
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		log.Error("Can't parse access token", slog.String("token", accessToken))
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.Err("invalid authorization token"))
		return
	}
	userID := claims["id"].(string)

	ctx.Set("UserID", userID)
	ctx.Next()
}
