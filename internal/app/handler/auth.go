package handler

import (
	"api/internal/app/handler/request/requestbody"
	"api/internal/app/handler/response"
	"api/internal/app/handler/response/responsebody"
	"api/internal/lib/sl"
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
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Message("invalid request body"))
		return
	}

	_, err := mail.ParseAddress(body.Email)
	if err != nil {
		log.Info("Email is invalid", slog.String("email", body.Email))
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Message("invalid email format"))
		return
	}

	if len(body.Name) > 50 {
		log.Info("Name is too long")
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Message("name is too long"))
		return
	}

	if len(body.Password) > 50 {
		log.Info("Password is too long")
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Message("password is too long"))
		return
	}

	user, err := h.repository.User.Create(c, body.Email, body.Name, sha256.String(body.Password))
	if errors.Is(err, repoerr.ErrUserAlreadyExists) {
		log.Info("User already exists", sl.Err(err))
		c.AbortWithStatusJSON(http.StatusConflict, response.Message("user already exists"))
		return
	}
	if err != nil {
		log.Error("Can't create user", sl.Err(err))
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Message("can't register"))
		return
	}

	log.Info("Created a user", slog.String("id", user.ID), slog.String("email", user.Email), slog.String("name", user.Name))

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
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Message("invalid request body"))
		return
	}

	user, err := h.repository.User.GetByCredentials(c, body.Email, sha256.String(body.Password))
	if errors.Is(err, repoerr.ErrUserNotFound) {
		log.Info("User not found", slog.String("email", body.Email))
		c.AbortWithStatusJSON(http.StatusNotFound, response.Message("user not found"))
		return
	}
	if err != nil {
		log.Error("Can't find user", sl.Err(err))
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Message("can't login"))
		return
	}

	token, err := h.token.GenerateToken(user.ID)
	if err != nil {
		log.Error("Can't generate JWT", sl.Err(err))
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Message("can't login"))
		return
	}

	c.JSON(http.StatusOK, responsebody.Token{
		Token: token,
	})
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
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Message("invalid request body"))
		return
	}

	_, err := h.repository.User.GetByEmail(c, body.Email)
	if errors.Is(err, repoerr.ErrUserNotFound) {
		log.Info("User not found", slog.String("email", body.Email))
		c.AbortWithStatusJSON(http.StatusNotFound, response.Message("user not found"))
		return
	}
	if err != nil {
		log.Error("Can't find user", sl.Err(err))
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Message("can't request password reset"))
		return
	}

	token := random.String(64)
	err = h.repository.User.CreatePasswordResetRequest(c, token, body.Email)
	if err != nil {
		log.Error("Can't save password reset request information", sl.Err(err))
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Message("can't request password reset"))
		return
	}

	err = h.mailer.SendRecoveryEmail(body.Email, token)
	if err != nil {
		log.Error("Can't send password reset link", sl.Err(err))
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Message("can't request password reset"))
		return
	}

	c.Status(http.StatusOK)
}

// @Summary      Update password
// @Description  updates password for user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input body            requestbody.UpdatePassword true "User information"
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
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Message("invalid request body"))
		return
	}

	passwordResetRequest, err := h.repository.User.GetPasswordResetRequestByToken(c, body.Token)
	if errors.Is(err, repoerr.ErrPasswordResetRequestNotFound) {
		log.Info("Password reset request not found", slog.String("token", body.Token))
		c.AbortWithStatusJSON(http.StatusNotFound, response.Message("password reset request not found"))
		return
	}
	if err != nil {
		log.Error("Can't get password reset request by token", sl.Err(err), slog.String("token", body.Token))
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Message("invalid request body"))
		return
	}

	if time.Now().After(passwordResetRequest.ExpiresAt) {
		log.Info("Password reset token already expired")
		c.AbortWithStatusJSON(http.StatusForbidden, response.Message("recovery token expired"))
		return
	}

	if passwordResetRequest.IsUsed {
		log.Info("Password reset token already used")
		c.AbortWithStatusJSON(http.StatusForbidden, response.Message("this recovery token has been used"))
		return
	}

	err = h.repository.User.UpdatePasswordByEmail(c, passwordResetRequest.Email, sha256.String(body.Password))
	if err != nil {
		log.Error("Can't update password", sl.Err(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Message("can't update password"))
		return
	}

	err = h.repository.User.MarkResetPasswordTokenAsUsed(c, passwordResetRequest.Token)
	if err != nil {
		log.Error("Can't mark token as used", sl.Err(err))
	}

	c.Status(http.StatusOK)
}
