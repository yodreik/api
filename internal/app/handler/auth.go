package handler

import (
	"api/internal/app/handler/request/requestbody"
	"api/internal/app/handler/response"
	"api/internal/app/handler/response/responsebody"
	"api/internal/lib/sl"
	repoerr "api/internal/repository/errors"
	"api/pkg/requestid"
	"api/pkg/sha256"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// @Summary      Register user
// @Description  create user in database
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input body      requestbody.Register true "User information"
// @Success      201 {object}    responsebody.User
// @Failure      400 {object}    responsebody.Error
// @Failure      409 {object}    responsebody.Error
// @Router       /auth/register  [post]
func (h *Handler) Register(ctx *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.Register"),
		slog.String("request_id", requestid.Get(ctx)),
	)

	var body requestbody.Register
	if err := ctx.BindJSON(&body); err != nil {
		log.Info("Can't decode request body", sl.Err(err))
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.Err("invalid request body"))
		return
	}

	_, err := mail.ParseAddress(body.Email)
	if err != nil {
		log.Info("Email is invalid", slog.String("email", body.Email))
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.Err("invalid email format"))
		return
	}

	if len(body.Name) > 50 {
		log.Info("Name is too long")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.Err("name is too long"))
		return
	}

	if len(body.Password) > 50 {
		log.Info("Password is too long")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.Err("password is too long"))
		return
	}

	user, err := h.repository.User.Create(ctx, body.Email, body.Name, sha256.String(body.Password))
	if errors.Is(err, repoerr.ErrUserAlreadyExists) {
		log.Info("User already exists", sl.Err(err))
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

// @Summary      Log into user's account
// @Description  check if user exists, and return an access token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input body    requestbody.Login true "User information"
// @Success      200 {object}  responsebody.Token
// @Failure      400 {object}  responsebody.Error
// @Failure      404 {object}  responsebody.Error
// @Router       /auth/login   [post]
func (h *Handler) Login(ctx *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.Login"),
		slog.String("request_id", requestid.Get(ctx)),
	)

	var body requestbody.Login
	if err := ctx.BindJSON(&body); err != nil {
		log.Info("Can't decode request body", sl.Err(err))
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.Err("invalid request body"))
		return
	}

	user, err := h.repository.User.GetByCredentials(ctx, body.Email, sha256.String(body.Password))
	if errors.Is(err, repoerr.ErrUserNotFound) {
		log.Info("User not found", slog.String("email", body.Email))
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

// @Summary      Get information about current user
// @Description  returns an user's information, that currently logged in
// @Security     AccessToken
// @Tags         auth
// @Produce      json
// @Success      200 {object}  responsebody.User
// @Failure      400 {object}  responsebody.Error
// @Failure      404 {object}  responsebody.Error
// @Router       /auth/me      [get]
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
