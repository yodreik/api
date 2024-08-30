package handler

import (
	"api/internal/app/handler/response"
	"api/internal/lib/sl"
	"api/pkg/requestid"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
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

	token := parts[1]

	userID, err := h.token.ParseToID(token)
	if err != nil {
		log.Error("Can't parse access token", slog.String("token", token), sl.Err(err))
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.Err("invalid authorization token"))
		return
	}

	ctx.Set("UserID", userID)
	ctx.Next()
}
