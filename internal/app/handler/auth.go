package handler

import (
	"api/internal/app/response"
	"api/internal/lib/sl"
	"api/pkg/requestid"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net/http"
	"net/mail"

	"github.com/gin-gonic/gin"
)

// TODO: Test this handlers

func (h *Handler) Register(ctx *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.Register"),
		slog.String("request_id", requestid.Get(ctx)),
	)

	// TODO: Move body struct to other package
	var body struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}

	if err := ctx.BindJSON(&body); err != nil {
		log.Error("Can't decode request body", sl.Err(err))
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.Err("invalid request body"))
		return
	}

	_, err := mail.ParseAddress(body.Email)
	if err != nil {
		log.Debug("Email is invalid", slog.String("email", body.Email))
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.Err("invalid email"))
		return
	}

	passwordHash := sha256.New()
	passwordHash.Write([]byte(body.Password))

	// TODO: Add handling of duplicate key error (= user already exists)
	user, err := h.repository.User.Create(ctx, body.Email, body.Name, hex.EncodeToString(passwordHash.Sum(nil)))
	if err != nil {
		log.Error("Can't create user", sl.Err(err))
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, response.Err("can't create user"))
		return
	}

	log.Info("Created a user", slog.String("id", user.ID), slog.String("email", user.Email), slog.String("name", user.Name))

	ctx.JSON(http.StatusCreated, response.User{
		ID:    user.ID,
		Email: body.Email,
		Name:  body.Name,
	})
}

func (h *Handler) Login(ctx *gin.Context) {}
