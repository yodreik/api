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
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// @Summary      Request password reset
// @Description  sends an email with recovery link
// @Tags         account
// @Accept       json
// @Produce      json
// @Param        input body                       requestbody.ResetPassword true "User information"
// @Success      200
// @Failure      400 {object}                     responsebody.Message
// @Failure      404 {object}                     responsebody.Message
// @Router       /account/reset-password/request  [post]
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
// @Tags         account
// @Accept       json
// @Produce      json
// @Param        input body               requestbody.UpdatePassword true "User information"
// @Success      200
// @Failure      400 {object}             responsebody.Message
// @Failure      404 {object}             responsebody.Message
// @Router       /account/reset-password  [patch]
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
// @Tags         account
// @Accept       json
// @Produce      json
// @Param        input body        requestbody.ConfirmAccount true "Token"
// @Success      200
// @Failure      400 {object}      responsebody.Message
// @Failure      404 {object}      responsebody.Message
// @Router       /account/confirm  [post]
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
// @Tags         account
// @Produce      json
// @Success      200 {object}  responsebody.Account
// @Failure      401 {object}  responsebody.Message
// @Router       /account      [get]
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
// @Tags         account
// @Accept       json
// @Produce      json
// @Param        input body    requestbody.UpdateAccount true "User Information"
// @Success      200
// @Failure      400 {object}  responsebody.Message
// @Failure      401 {object}  responsebody.Message
// @Router       /account      [patch]
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

	if body.Email != nil {
		user.Email = *body.Email
		user.IsConfirmed = false
		user.ConfirmationToken = uuid.NewString()

		go func() {
			err = h.mailer.SendConfirmationEmail(user.Email, user.ConfirmationToken)
			if err != nil {
				log.Error("can't send an email", sl.Err(err))
			}
		}()
	}
	if body.Username != nil {
		user.Username = *body.Username
	}
	if body.DisplayName != nil {
		user.DisplayName = *body.DisplayName
	}
	if body.Password != nil {
		user.PasswordHash = sha256.String(*body.Password)
	}
	if body.IsPrivate != nil {
		user.IsPrivate = *body.IsPrivate
	}

	err = h.repository.User.UpdateUser(c, userID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate, user.IsConfirmed, user.ConfirmationToken)
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
// @Tags         account
// @Accept       multipart/form-data
// @Produce      json
// @Param        avatar formData  file true "Avatar Image"
// @Success      200
// @Failure      400 {object}     responsebody.Message
// @Failure      404 {object}     responsebody.Message
// @Router       /account/avatar  [patch]
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

	maxFileSize := int64(1024 * 1024 * 2) // 2Mb
	if file.Size > maxFileSize {
		log.Debug("file too big", slog.Int64("size", file.Size))
		response.WithMessage(c, http.StatusBadRequest, "file should be smaller than 2Mb")
		return
	}

	extension := filepath.Ext(file.Filename)
	if extension != ".png" && extension != ".jpg" && extension != ".jpeg" {
		log.Debug("invalid extension", slog.String("extension", extension))
		response.WithMessage(c, http.StatusBadRequest, "only png, jpg and jpeg files are available")
		return
	}

	filename := fmt.Sprintf("%s%s", uuid.NewString(), extension)

	dst := fmt.Sprintf("./.database/avatars/%s", filename)

	// Generate new filename, until it isn't taken
	_, err = os.Stat(dst)
	for err == nil {
		filename = fmt.Sprintf("%s%s", uuid.NewString(), extension)
		dst = fmt.Sprintf("./.database/avatars/%s", filename)

		_, err = os.Stat(dst)
	}

	err = c.SaveUploadedFile(file, dst)
	if err != nil {
		log.Error("could not save file", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	avatarParts := strings.Split(user.AvatarURL, "/")
	if len(avatarParts) > 0 {
		prevAvatarPath := avatarParts[len(avatarParts)-1]
		err := os.Remove(fmt.Sprintf("./.database/avatars/%s", prevAvatarPath))
		if err != nil {
			log.Debug("can't remove old avatar file", sl.Err(err))
		}
	}

	user.AvatarURL = fmt.Sprintf("%s/api/avatar/%s", h.config.BasePath, filename)

	err = h.repository.User.UpdateUser(c, user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate, user.IsConfirmed, user.ConfirmationToken)
	if err != nil {
		log.Error("could not update user", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	c.Status(http.StatusOK)
}

// @Summary      Delete user avatar
// @Description  deletes user's avatar image
// @Security     AccessToken
// @Tags         account
// @Produce      json
// @Success      200
// @Failure      404 {object}     responsebody.Message
// @Router       /account/avatar  [delete]
func (h *Handler) DeleteAvatar(c *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.DeleteAvatar"),
		slog.String("request_id", requestid.Get(c)),
	)

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

	user.AvatarURL = ""

	err = h.repository.User.UpdateUser(c, user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate, user.IsConfirmed, user.ConfirmationToken)
	if err != nil {
		log.Error("could not update user", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	c.Status(http.StatusOK)
}
