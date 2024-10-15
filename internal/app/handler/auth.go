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
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"net/url"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
	})
}

// @Summary      Update personal information
// @Description  updates user entity in storage
// @Security     AccessToken
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input body             requestbody.UpdateAccount true "User Information"
// @Success      200
// @Failure      400 {object}   responsebody.Message
// @Failure      401 {object}   responsebody.Message
// @Router       /auth/account  [patch]
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
		_, err = url.ParseRequestURI(*body.AvatarURL)
		if err != nil {
			log.Debug("invalid avatar url link", slog.String("avatar_url", *body.AvatarURL))
			response.WithMessage(c, http.StatusBadRequest, "avatar_url should be a valid link")
			return
		}

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

// @Summary      Upload User Avatar
// @Description  uploads a new avatar image for the user. Only PNG, JPG, and JPEG formats are allowed
// @Security     AccessToken
// @Tags         auth
// @Accept       multipart/form-data
// @Produce      json
// @Param        avatar formData       file true "Avatar Image"
// @Success      200 {object}          responsebody.Account
// @Failure      400 {object}          responsebody.Message
// @Failure      404 {object}          responsebody.Message
// @Router       /auth/account/avatar  [patch]
func (h *Handler) UploadAvatar(c *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.UploadAvatar"),
		slog.String("request_id", requestid.Get(c)),
	)

	// TODO: Return error if userID is empty
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

	file, err := c.FormFile("avatar")
	if err != nil {
		log.Error("can't get file form", sl.Err(err))
		response.WithMessage(c, http.StatusBadRequest, "no avatar image provided")
		return
	}

	extension := filepath.Ext(file.Filename)
	if extension != ".png" && extension != ".jpg" && extension != ".jpeg" {
		log.Debug("invalid extension", slog.String("extension", extension))
		response.WithMessage(c, http.StatusBadRequest, "only png, jpg and jpeg files are available")
		return
	}

	// TODO: Check if file is too big

	filename := fmt.Sprintf("%s%s", uuid.NewString(), extension)

	dst := fmt.Sprintf("./.database/avatars/%s", filename)

	// TODO: Check if file with this name already exists
	err = c.SaveUploadedFile(file, dst)
	if err != nil {
		log.Error("could not save file", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	// TODO: Take basepath for avatar from config
	user.AvatarURL = fmt.Sprintf("https://dreik.d.qarwe.online/api/avatar/%s", filename)

	err = h.repository.User.UpdateUser(c, user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate)
	if err != nil {
		log.Error("could not update user", sl.Err(err))
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
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
	})
}
