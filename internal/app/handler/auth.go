package handler

import (
	"api/internal/app/handler/request/requestbody"
	"api/internal/app/handler/response"
	"api/internal/app/handler/response/responsebody"
	"api/internal/lib/logger/sl"
	repoerr "api/internal/repository/errors"
	"api/pkg/random"
	"api/pkg/requestid"
	"api/pkg/sha256"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"time"

	"github.com/gin-gonic/gin"
)

// @Summary      Register user
// @Description  create user in database
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input body      requestbody.Register true "User information"
// @Success      201 {object}    responsebody.User
// @Failure      400 {object}    responsebody.Message
// @Failure      403 {object}    responsebody.Message
// @Failure      409 {object}    responsebody.Message
// @Router       /auth/register  [post]
func (h *Handler) Register(c *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.Register"),
		slog.String("request_id", requestid.Get(c)),
	)

	var body requestbody.Register
	if err := c.BindJSON(&body); err != nil {
		log.Info("Can't decode request body", sl.Err(err))
		response.InvalidRequestBody(c)
		return
	}

	_, err := mail.ParseAddress(body.Email)
	if err != nil {
		log.Info("Email is invalid", slog.String("email", body.Email))
		response.WithMessage(c, http.StatusBadRequest, "invalid email format")
		return
	}

	if len(body.Name) > 50 {
		log.Info("Name is too long")
		response.WithMessage(c, http.StatusBadRequest, "name is too long")
		return
	}

	if len(body.Password) > 50 {
		log.Info("Password is too long")
		response.WithMessage(c, http.StatusBadRequest, "password is too long")
		return
	}

	token := h.token.Long()
	user, err := h.repository.User.CreateWithEmailConfirmationRequest(c, body.Email, body.Name, sha256.String(body.Password), token)
	if errors.Is(err, repoerr.ErrUserAlreadyExists) {
		log.Info("User already exists", sl.Err(err))
		response.WithMessage(c, http.StatusConflict, "user already exists")
		return
	}
	if err != nil {
		log.Error("Can't create user", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	log.Info("Created a user", slog.String("id", user.ID), slog.String("email", user.Email), slog.String("name", user.Name))

	err = h.mailer.SendConfirmationEmail(body.Email, token)
	if err != nil {
		log.Error("Can't send an email", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	// TOTHINK: Maybe additionally return an access token
	c.JSON(http.StatusCreated, responsebody.User{
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
// @Failure      400 {object}  responsebody.Message
// @Failure      404 {object}  responsebody.Message
// @Router       /auth/login   [post]
func (h *Handler) Login(c *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.Login"),
		slog.String("request_id", requestid.Get(c)),
	)

	var body requestbody.Login
	if err := c.BindJSON(&body); err != nil {
		log.Info("Can't decode request body", sl.Err(err))
		response.InvalidRequestBody(c)
		return
	}

	user, err := h.repository.User.GetByCredentials(c, body.Email, sha256.String(body.Password))
	if errors.Is(err, repoerr.ErrUserNotFound) {
		log.Info("User not found", slog.String("email", body.Email))
		response.WithMessage(c, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		log.Error("Can't find user", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	if user.IsEmailConfirmed {
		token, err := h.token.GenerateJWT(user.ID)
		if err != nil {
			log.Error("Can't generate JWT", sl.Err(err))
			response.InternalServerError(c)
			return
		}

		c.JSON(http.StatusOK, responsebody.Token{
			Token: token,
		})
		return
	}

	log.Info("User's email not confirmed")
	request, err := h.repository.User.GetRequestByEmail(c, body.Email)
	if errors.Is(err, repoerr.ErrRequestNotFound) || time.Now().After(request.ExpiresAt) {
		log.Info("Confirmation request not found")
		token := random.String(64)
		request, err := h.repository.User.CreateEmailConfirmationRequest(c, token, body.Email)
		if err != nil {
			log.Error("Can't save new email confirmation request", sl.Err(err))
			response.InternalServerError(c)
			return
		}

		err = h.mailer.SendConfirmationEmail(body.Email, request.Token)
		if err != nil {
			log.Error("Can't send confirmation email", sl.Err(err))
			response.InternalServerError(c)
			return
		}
		log.Info("New confirmation email sent")
		response.WithMessage(c, http.StatusForbidden, "email confirmation needed")
		return
	}
	if err != nil {
		log.Error("Can't get confirmation request", sl.Err(err))
		response.InternalServerError(c)
		return
	}

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
		log.Info("Can't decode request body", sl.Err(err))
		response.InvalidRequestBody(c)
		return
	}

	_, err := h.repository.User.GetByEmail(c, body.Email)
	if errors.Is(err, repoerr.ErrUserNotFound) {
		log.Info("User not found", slog.String("email", body.Email))
		response.WithMessage(c, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		log.Error("Can't find user", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	token := h.token.Long()
	request, err := h.repository.User.CreatePasswordResetRequest(c, token, body.Email)
	if err != nil {
		log.Error("Can't save password reset request information", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	err = h.mailer.SendRecoveryEmail(body.Email, request.Token)
	if err != nil {
		log.Error("Can't send password reset link", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	c.Status(http.StatusOK)
}

// @Summary      Update password
// @Description  updates password for user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input body             requestbody.UpdatePassword true "User information"
// @Success      200
// @Failure      400 {object}           responsebody.Message
// @Failure      404 {object}           responsebody.Message
// @Router       /auth/password/update  [patch]
func (h *Handler) UpdatePassword(c *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.UpdatePassword"),
		slog.String("request_id", requestid.Get(c)),
	)

	var body requestbody.UpdatePassword
	if err := c.BindJSON(&body); err != nil {
		log.Info("Can't decode request body", sl.Err(err))
		response.InvalidRequestBody(c)
		return
	}

	passwordResetRequest, err := h.repository.User.GetRequestByToken(c, body.Token)
	if errors.Is(err, repoerr.ErrRequestNotFound) {
		log.Info("Password reset request not found", slog.String("token", body.Token))
		response.WithMessage(c, http.StatusNotFound, "password reset request not found")
		return
	}
	if err != nil {
		log.Error("Can't get password reset request by token", sl.Err(err), slog.String("token", body.Token))
		response.InternalServerError(c)
		return
	}

	if time.Now().After(passwordResetRequest.ExpiresAt) {
		log.Info("Password reset token already expired")
		response.WithMessage(c, http.StatusForbidden, "recovery token expired")
		return
	}

	if passwordResetRequest.IsUsed {
		log.Info("Password reset token already used")
		response.WithMessage(c, http.StatusForbidden, "this recovery token has been used")
		return
	}

	err = h.repository.User.UpdatePasswordByEmail(c, passwordResetRequest.Email, sha256.String(body.Password))
	if err != nil {
		log.Error("Can't update password", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	err = h.repository.User.MarkRequestAsUsed(c, passwordResetRequest.Token)
	if err != nil {
		log.Error("Can't mark token as used", sl.Err(err))
	}

	c.Status(http.StatusOK)
}

// @Summary      Confirm email
// @Description  confirms user's email
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input body             requestbody.ConfirmEmail true "Token"
// @Success      200
// @Failure      400 {object}           responsebody.Message
// @Failure      404 {object}           responsebody.Message
// @Router       /auth/confirm          [post]
func (h *Handler) ConfirmEmail(c *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.ConfirmEmail"),
		slog.String("request_id", requestid.Get(c)),
	)

	var body requestbody.ConfirmEmail
	if err := c.BindJSON(&body); err != nil {
		log.Info("Can't decode request body", sl.Err(err))
		response.InvalidRequestBody(c)
		return
	}

	request, err := h.repository.User.GetRequestByToken(c, body.Token)
	if errors.Is(err, repoerr.ErrRequestNotFound) {
		log.Error("Request not found")
		response.WithMessage(c, http.StatusNotFound, "confirmation request not found")
		return
	}
	if err != nil {
		log.Error("Can't confirm email", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	if time.Now().After(request.ExpiresAt) {
		log.Info("Confirmation token expired")
		err := h.mailer.SendConfirmationEmail(request.Email, request.Token)
		if err != nil {
			log.Error("Can't send confirmation email", sl.Err(err))
			response.InternalServerError(c)
			return
		}
		response.WithMessage(c, http.StatusForbidden, "confirmation link expired. we will send you new confirmation email")
		return
	}

	err = h.repository.User.ConfirmEmail(c, request.Email, request.Token)
	if err != nil {
		log.Error("Can't confirm email", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	c.Status(http.StatusOK)
}
