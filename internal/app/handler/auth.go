package handler

import (
	requestbody "api/internal/app/handler/request/body"
	"api/internal/app/handler/response"
	responsebody "api/internal/app/handler/response/body"
	"api/internal/lib/sl"
	repoerr "api/internal/repository/errors"
	"api/pkg/requestid"
	"api/pkg/sha256"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// TODO: Test this handlers

func (h *Handler) Register(ctx *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.Register"),
		slog.String("request_id", requestid.Get(ctx)),
	)

	var body requestbody.Register

	if err := ctx.BindJSON(&body); err != nil {
		log.Debug("Can't decode request body", sl.Err(err))
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.Err("invalid request body"))
		return
	}

	_, err := mail.ParseAddress(body.Email)
	if err != nil {
		log.Debug("Email is invalid", slog.String("email", body.Email))
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.Err("invalid email format"))
		return
	}

	user, err := h.repository.User.Create(ctx, body.Email, body.Name, sha256.String(body.Password))
	if errors.Is(err, repoerr.ErrUserAlreadyExists) {
		log.Debug("User already exists", sl.Err(err))
		ctx.AbortWithStatusJSON(http.StatusConflict, response.Err("user already exists"))
		return
	}
	if err != nil {
		log.Error("Can't create user", sl.Err(err))
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, response.Err("can't register"))
		return
	}

	log.Info("Created a user", slog.String("id", user.ID), slog.String("email", user.Email), slog.String("name", user.Name))

	// TOTHINK: Maybe additionally return an access token
	ctx.JSON(http.StatusCreated, responsebody.User{
		ID:    user.ID,
		Email: body.Email,
		Name:  body.Name,
	})
}

func (h *Handler) Login(ctx *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.Login"),
		slog.String("request_id", requestid.Get(ctx)),
	)

	var body requestbody.Login

	if err := ctx.BindJSON(&body); err != nil {
		log.Debug("Can't decode request body", sl.Err(err))
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.Err("invalid request body"))
		return
	}

	user, err := h.repository.User.GetByCredentials(ctx, body.Email, sha256.String(body.Password))
	if errors.Is(err, repoerr.ErrUserNotFound) {
		log.Debug("User not found", slog.String("email", body.Email))
		ctx.AbortWithStatusJSON(http.StatusNotFound, response.Err("user not found"))
		return
	}
	if err != nil {
		log.Error("Can't find user", sl.Err(err))
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, response.Err("can't login"))
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iat": time.Now().Unix(),
		"id":  user.ID,
	})

	tokenString, err := token.SignedString([]byte(h.config.Token.Secret))
	if err != nil {
		log.Error("Can't generate JWT", sl.Err(err))
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, response.Err("can't login"))
		return
	}

	ctx.JSON(http.StatusOK, responsebody.Token{
		Token: tokenString,
	})
}
