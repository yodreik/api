package handler

import (
	"api/internal/app/handler/request/requestbody"
	"api/internal/app/handler/response"
	"api/internal/app/handler/response/responsebody"
	"api/internal/lib/logger/sl"
	"api/internal/repository/entity"
	repoerr "api/internal/repository/errors"
	"api/pkg/requestid"
	"api/pkg/sha256"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"time"

	"github.com/gin-gonic/gin"
)

// @Summary      Create new account
// @Description  create user in database
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input body     requestbody.CreateAccount true "User information"
// @Success      201 {object}   responsebody.Account
// @Failure      400 {object}   responsebody.Message
// @Failure      409 {object}   responsebody.Message
// @Router       /auth/account  [post]
func (h *Handler) CreateAccount(c *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.CreateAccount"),
		slog.String("request_id", requestid.Get(c)),
	)

	var body requestbody.CreateAccount
	if err := c.BindJSON(&body); err != nil {
		log.Debug("can't decode request body", sl.Err(err))
		response.InvalidRequestBody(c)
		return
	}

	_, err := mail.ParseAddress(body.Email)
	if err != nil {
		log.Debug("email is invalid", slog.String("email", body.Email))
		response.WithMessage(c, http.StatusBadRequest, "invalid email format")
		return
	}

	user, err := h.repository.User.Create(c, body.Email, body.Username, sha256.String(body.Password))
	if errors.Is(err, repoerr.ErrUserAlreadyExists) {
		log.Info("user already exists", sl.Err(err))
		response.WithMessage(c, http.StatusConflict, "user already exists")
		return
	}
	if err != nil {
		log.Error("can't create user", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	log.Info("created a user", slog.String("id", user.ID), slog.String("email", user.Email), slog.String("username", user.Username))

	go func() {
		err = h.mailer.SendConfirmationEmail(body.Email, user.ConfirmationToken)
		if err != nil {
			log.Error("can't send an email", sl.Err(err))
		}
	}()

	c.JSON(http.StatusCreated, responsebody.Account{
		ID:          user.ID,
		Email:       user.Email,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		IsPrivate:   user.IsPrivate,
		IsConfirmed: user.IsConfirmed,
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
	})
}

// @Summary      Create a session for existing account
// @Description  check if user exists, and return an access token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input body     requestbody.CreateSession true "User information"
// @Success      200 {object}   responsebody.Token
// @Failure      400 {object}   responsebody.Message
// @Failure      401 {object}   responsebody.Message
// @Router       /auth/session  [post]
func (h *Handler) CreateSession(c *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.CreateSession"),
		slog.String("request_id", requestid.Get(c)),
	)

	var body requestbody.CreateSession
	if err := c.BindJSON(&body); err != nil {
		log.Debug("can't decode request body", sl.Err(err))
		response.InvalidRequestBody(c)
		return
	}

	var err error
	var user *entity.User
	if _, mailErr := mail.ParseAddress(body.Login); mailErr == nil {
		user, err = h.repository.User.GetByCredentialsWithEmail(c, body.Login, sha256.String(body.Password))
	} else {
		user, err = h.repository.User.GetByCredentialsWithUsername(c, body.Login, sha256.String(body.Password))
	}

	if errors.Is(err, repoerr.ErrUserNotFound) {
		log.Debug("user not found", slog.String("login", body.Login))
		response.WithMessage(c, http.StatusUnauthorized, "user not found")
		return
	}
	if err != nil {
		log.Error("can't find user", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	if user.IsConfirmed {
		token, err := h.token.GenerateJWT(user.ID)
		if err != nil {
			log.Error("can't generate JWT", sl.Err(err))
			response.InternalServerError(c)
			return
		}

		c.JSON(http.StatusOK, responsebody.Token{
			Token: token,
		})
		return
	}

	log.Debug("user's email not confirmed")

	go func() {
		err = h.mailer.SendConfirmationEmail(user.Email, user.ConfirmationToken)
		if err != nil {
			log.Error("can't send confirmation email", sl.Err(err))
		}
	}()

	response.WithMessage(c, http.StatusForbidden, "email confirmation needed")
}
