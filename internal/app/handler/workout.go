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
// @Failure      400 {object}  responsebody.Message
// @Router       /workout      [post]
func (h *Handler) CreateWorkout(c *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.CreateWorkout"),
		slog.String("request_id", requestid.Get(c)),
	)

	var body requestbody.CreateWorkout
	if err := c.BindJSON(&body); err != nil {
		log.Info("Can't decode request body", sl.Err(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Message("invalid request body"))
		return
	}

	date, err := time.Parse("02.01.2006", body.Date)
	if err != nil {
		log.Info("Invalid date format", sl.Err(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Message("invalid date format"))
		return
	}

	userID := c.GetString("UserID")
	workout, err := h.repository.Workout.Create(c, userID, date, body.Duration, body.Kind)
	if err != nil {
		log.Error("Can't create workout", sl.Err(err))
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Message("can't create workout record"))
		return
	}

	log.Info("Created a workout record", slog.String("id", workout.ID))

	c.JSON(http.StatusCreated, responsebody.Workout{
		ID:       workout.ID,
		Date:     date.Format("02.01.2006"),
		Duration: workout.Duration,
		Kind:     workout.Kind,
	})
}
