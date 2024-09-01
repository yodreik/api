package handler

import (
	"api/internal/app/handler/request/requestbody"
	"api/internal/app/handler/response"
	"api/internal/app/handler/response/responsebody"
	"api/internal/lib/sl"
	"api/pkg/requestid"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// @Summary      Create a record about past workout
// @Description  creates a new record about workout session
// @Security     AccessToken
// @Tags         workout
// @Accept       json
// @Produce      json
// @Param        input body    requestbody.CreateWorkout true "Information about workout session"
//
// @Success      201 {object}  responsebody.Workout
// @Failure      400 {object}  responsebody.Error
// @Router       /workout      [post]
func (h *Handler) CreateWorkout(ctx *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.CreateWorkout"),
		slog.String("request_id", requestid.Get(ctx)),
	)

	var body requestbody.CreateWorkout
	if err := ctx.BindJSON(&body); err != nil {
		log.Info("Can't decode request body", sl.Err(err))
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.Err("invalid request body"))
		return
	}

	date, err := time.Parse("02.01.2006", body.Date)
	if err != nil {
		log.Info("Invalid date format", sl.Err(err))
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.Err("invalid date format"))
		return
	}

	userID := ctx.GetString("UserID")
	workout, err := h.repository.Workout.Create(ctx, userID, date, body.Duration, body.Kind)
	if err != nil {
		log.Error("Can't create workout", sl.Err(err))
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, response.Err("can't register"))
		return
	}

	log.Info("Created a workout record", slog.String("id", workout.ID))

	ctx.JSON(http.StatusCreated, responsebody.Workout{
		ID:       workout.ID,
		Date:     date.Format("02.01.2006"),
		Duration: workout.Duration,
		Kind:     workout.Kind,
	})
}
