package handler

import (
	"api/internal/app/handler/request/requestbody"
	"api/internal/app/handler/response"
	"api/internal/app/handler/response/responsebody"
	"api/internal/lib/logger/sl"
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

	if len(body.Username) < 5 {
		log.Debug("username is too short")
		response.WithMessage(c, http.StatusBadRequest, "username is too short")
		return
	}

	if len(body.Password) > 50 {
		log.Debug("password is too long")
		response.WithMessage(c, http.StatusBadRequest, "password is too long")
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

	user, err := h.repository.User.GetByCredentials(c, body.Email, sha256.String(body.Password))
	if errors.Is(err, repoerr.ErrUserNotFound) {
		log.Debug("user not found", slog.String("email", body.Email))
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
		err = h.mailer.SendConfirmationEmail(body.Email, user.ConfirmationToken)
		if err != nil {
			log.Error("can't send confirmation email", sl.Err(err))
		}
	}()

	response.WithMessage(c, http.StatusForbidden, "email confirmation needed")
}

// @Summary      Request password reset
// @Description  sends an email with recovery link
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input body            requestbody.ResetPassword true "User information"
// @Success      200
// @Failure      400 {object}          responsebody.Message
// @Failure      404 {object}          responsebody.Message
// @Router       /auth/password/reset  [post]
func (h *Handler) ResetPassword(c *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.ResetPassword"),
		slog.String("request_id", requestid.Get(c)),
	)

	var body requestbody.ResetPassword
	if err := c.BindJSON(&body); err != nil {
		log.Debug("can't decode request body", sl.Err(err))
		response.InvalidRequestBody(c)
		return
	}

	_, err := h.repository.User.GetByEmail(c, body.Email)
	if errors.Is(err, repoerr.ErrUserNotFound) {
		log.Debug("user not found", slog.String("email", body.Email))
		response.WithMessage(c, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		log.Error("can't find user", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	token := h.token.Long()
	request, err := h.repository.User.CreatePasswordResetRequest(c, token, body.Email)
	if err != nil {
		log.Error("can't save password reset request information", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	go func() {
		err = h.mailer.SendRecoveryEmail(body.Email, request.Token)
		if err != nil {
			log.Error("can't send password reset link", sl.Err(err))
		}
	}()

	c.Status(http.StatusOK)
}

// @Summary      Update password
// @Description  updates password for user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input body      requestbody.UpdatePassword true "User information"
// @Success      200
// @Failure      400 {object}    responsebody.Message
// @Failure      404 {object}    responsebody.Message
// @Router       /auth/password  [patch]
func (h *Handler) UpdatePassword(c *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.UpdatePassword"),
		slog.String("request_id", requestid.Get(c)),
	)

	var body requestbody.UpdatePassword
	if err := c.BindJSON(&body); err != nil {
		log.Debug("can't decode request body", sl.Err(err))
		response.InvalidRequestBody(c)
		return
	}

	passwordResetRequest, err := h.repository.User.GetRequestByToken(c, body.Token)
	if errors.Is(err, repoerr.ErrRequestNotFound) {
		log.Debug("password reset request not found", slog.String("token", body.Token))
		response.WithMessage(c, http.StatusNotFound, "password reset request not found")
		return
	}
	if err != nil {
		log.Error("can't get password reset request by token", sl.Err(err), slog.String("token", body.Token))
		response.InternalServerError(c)
		return
	}

	if time.Now().After(passwordResetRequest.ExpiresAt) {
		log.Debug("password reset token already expired")
		response.WithMessage(c, http.StatusForbidden, "recovery token expired")
		return
	}

	if passwordResetRequest.IsUsed {
		log.Debug("password reset token already used")
		response.WithMessage(c, http.StatusForbidden, "this recovery token has been used")
		return
	}

	err = h.repository.User.UpdatePasswordByEmail(c, passwordResetRequest.Email, sha256.String(body.Password))
	if err != nil {
		log.Error("can't update password", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	err = h.repository.User.MarkRequestAsUsed(c, passwordResetRequest.Token)
	if err != nil {
		log.Error("can't mark token as used", sl.Err(err))
	}

	c.Status(http.StatusOK)
}

// @Summary      Confirm account's email
// @Description  confirms user's email
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input body             requestbody.ConfirmAccount true "Token"
// @Success      200
// @Failure      400 {object}           responsebody.Message
// @Failure      404 {object}           responsebody.Message
// @Router       /auth/account/confirm  [post]
func (h *Handler) ConfirmAccount(c *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.ConfirmAccount"),
		slog.String("request_id", requestid.Get(c)),
	)

	var body requestbody.ConfirmAccount
	if err := c.BindJSON(&body); err != nil {
		log.Debug("can't decode request body", sl.Err(err))
		response.InvalidRequestBody(c)
		return
	}

	user, err := h.repository.User.GetByConfirmationToken(c, body.Token)
	if errors.Is(err, repoerr.ErrUserNotFound) {
		log.Error("user not found")
		response.WithMessage(c, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		log.Error("can't confirm email", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	err = h.repository.User.SetUserConfirmed(c, user.Email, user.ConfirmationToken)
	if err != nil {
		log.Error("can't mark user as confirmed", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	c.Status(http.StatusOK)
}

// @Summary      Get information about current user
// @Description  returns an user's information, that currently logged in
// @Security     AccessToken
// @Tags         auth
// @Produce      json
// @Success      200 {object}   responsebody.Account
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

	c.JSON(http.StatusOK, responsebody.Account{
		ID:          user.ID,
		Email:       user.Email,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		AvatarURL:   user.AvatarURL,
		IsPrivate:   user.IsPrivate,
		IsConfirmed: user.IsConfirmed,
	})
}

func (h *Handler) UpdateAccount(c *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.UpdateAccount"),
		slog.String("request_id", requestid.Get(c)),
	)

	var body requestbody.UpdateAccount
	if err := c.BindJSON(&body); err != nil {
		log.Debug("can't decode request body", sl.Err(err))
		response.InvalidRequestBody(c)
		return
	}

	userID := c.GetString("UserID")
	user, err := h.repository.User.GetByID(c, userID)
	if errors.Is(err, repoerr.ErrUserNotFound) {
		log.Debug("user does not exists")
		response.WithMessage(c, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		log.Error("can't find user", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	if body.Username != nil {
		user.Username = *body.Username
	}
	if body.DisplayName != nil {
		user.DisplayName = *body.DisplayName
	}
	if body.AvatarURL != nil {
		user.AvatarURL = *body.AvatarURL
	}
	if body.Password != nil {
		user.PasswordHash = sha256.String(*body.Password)
	}
	if body.IsPrivate != nil {
		user.IsPrivate = *body.IsPrivate
	}

	err = h.repository.User.UpdateUser(c, userID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate)
	if err != nil {
		log.Error("can't update user", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	c.Status(http.StatusOK)
}
