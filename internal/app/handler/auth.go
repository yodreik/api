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

	"github.com/gin-gonic/gin"
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

	token, err := h.token.GenerateToken(user.ID)
	if err != nil {
		log.Error("Can't generate JWT", sl.Err(err))
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, response.Err("can't login"))
		return
	}

	ctx.JSON(http.StatusOK, responsebody.Token{
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
// @Failure      400 {object}          responsebody.Error
// @Failure      404 {object}          responsebody.Error
// @Router       /auth/password/reset  [post]
func (h *Handler) ResetPassword(ctx *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.ResetPassword"),
		slog.String("request_id", requestid.Get(ctx)),
	)

	var body requestbody.ResetPassword
	if err := ctx.BindJSON(&body); err != nil {
		log.Info("Can't decode request body", sl.Err(err))
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.Err("invalid request body"))
		return
	}

	_, err := h.repository.User.GetByEmail(ctx, body.Email)
	if errors.Is(err, repoerr.ErrUserNotFound) {
		log.Info("User not found", slog.String("email", body.Email))
		ctx.AbortWithStatusJSON(http.StatusNotFound, response.Err("user not found"))
		return
	}
	if err != nil {
		log.Error("Can't find user", sl.Err(err))
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, response.Err("can't request password reset"))
		return
	}

	token := random.String(64)
	err = h.repository.Cache.SetPasswordResetRequest(ctx, body.Email, token)
	if err != nil {
		log.Error("Can't save password reset request information", sl.Err(err))
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, response.Err("can't request password reset"))
		return
	}

	err = h.mailer.Send(body.Email, "welnex: Reset password", "This is youe token to reset password: "+token)
	if err != nil {
		log.Error("Can't send password reset link", sl.Err(err))
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, response.Err("can't request password reset"))
		return
	}

	ctx.Status(http.StatusOK)
}

// @Summary      Update password
// @Description  updates password for user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input body            requestbody.UpdatePassword true "User information"
// @Success      200
// @Failure      400 {object}           responsebody.Error
// @Failure      404 {object}           responsebody.Error
// @Router       /auth/password/update  [post]
func (h *Handler) UpdatePassword(ctx *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.UpdatePassword"),
		slog.String("request_id", requestid.Get(ctx)),
	)

	var body requestbody.UpdatePassword
	if err := ctx.BindJSON(&body); err != nil {
		log.Info("Can't decode request body", sl.Err(err))
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.Err("invalid request body"))
		return
	}

	email, err := h.repository.Cache.GetPasswordResetEmailByToken(ctx, body.Token)
	if errors.Is(err, repoerr.ErrPasswordResetRequestNotFound) {
		log.Info("Password reset request not found", slog.String("token", body.Token))
		ctx.AbortWithStatusJSON(http.StatusNotFound, response.Err("password reset request not found"))
		return
	}
	if err != nil {
		log.Error("Can't get password reset request by token", sl.Err(err), slog.String("token", body.Token))
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.Err("invalid request body"))
		return
	}

	err = h.repository.User.UpdatePasswordByEmail(ctx, email, sha256.String(body.Password))
	if err != nil {
		log.Error("Can't update password", sl.Err(err))
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.Err("can't update password"))
		return
	}

	ctx.Status(http.StatusOK)
}
