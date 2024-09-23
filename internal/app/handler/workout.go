package handler

import (
	"api/internal/app/handler/request/requestbody"
	"api/internal/app/handler/response"
	"api/internal/app/handler/response/responsebody"
	"api/internal/lib/logger/sl"
	"api/pkg/requestid"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// @Summary      Create a record about past workout
// @Description  creates a new record about workout session
// @Security     AccessToken
// @Tags         activity
// @Accept       json
// @Produce      json
// @Param        input body    requestbody.CreateWorkout true "Information about workout session"
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
		log.Debug("can't decode request body", sl.Err(err))
		response.InvalidRequestBody(c)
		return
	}

	layout := "02-01-2006"
	date, err := time.Parse(layout, body.Date)
	if err != nil {
		log.Debug("invalid date format", sl.Err(err))
		response.WithMessage(c, http.StatusBadRequest, "invalid date format")
		return
	}

	today, _ := time.Parse(layout, time.Now().Format(layout))
	if date.After(today) {
		log.Debug("workout record should be today or before", slog.String("date", date.Format(layout)))
		response.WithMessage(c, http.StatusBadRequest, "workout can't be in future")
		return
	}

	userID := c.GetString("UserID")
	workout, err := h.repository.Workout.Create(c, userID, date, body.Duration, body.Kind)
	if err != nil {
		log.Error("can't create workout", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	log.Info("created a workout record", slog.String("id", workout.ID))

	c.JSON(http.StatusCreated, responsebody.Workout{
		ID:       workout.ID,
		Date:     date.Format(layout),
		Duration: workout.Duration,
		Kind:     workout.Kind,
	})
}

// @Summary      Get user's activity history
// @Description  returns user's workout history
// @Security     AccessToken
// @Tags         activity
// @Accept       json
// @Produce      json
// @Param        begin query   string true "Begin date"
// @Param        end   query   string true "End date"
// @Success      200 {object}  responsebody.ActivityHistory
// @Failure      400 {object}  responsebody.Message
// @Failure      401 {object}  responsebody.Message
// @Router       /activity     [get]
func (h *Handler) GetActivityHistory(c *gin.Context) {
	log := slog.With(
		slog.String("op", "handler.GetActivityHistory"),
		slog.String("request_id", requestid.Get(c)),
	)

	params := c.Request.URL.Query()
	begin := params.Get("begin")
	end := params.Get("end")

	beginDate, err := time.Parse("02-01-2006", begin)
	if err != nil {
		log.Debug("incorrect date format", slog.String("date", begin), sl.Err(err))
		response.WithMessage(c, http.StatusBadRequest, "date not provided or invalid date format")
		return
	}

	endDate, err := time.Parse("02-01-2006", end)
	if err != nil {
		log.Debug("incorrect date format", slog.String("date", begin), sl.Err(err))
		response.WithMessage(c, http.StatusBadRequest, "date not provided or invalid date format")
		return
	}

	userID := c.GetString("UserID")
	workouts, err := h.repository.Workout.GetUserWorkouts(c, userID, beginDate, endDate)
	if err != nil {
		log.Error("can't get workouts", sl.Err(err))
		response.InternalServerError(c)
		return
	}

	res := responsebody.ActivityHistory{
		UserID:   userID,
		Count:    len(workouts),
		Workouts: make([]responsebody.Workout, 0),
	}

	for _, workout := range workouts {
		res.Workouts = append(res.Workouts, responsebody.Workout{
			ID:       workout.ID,
			Date:     workout.Date.Format("02-01-2006"),
			Duration: workout.Duration,
			Kind:     workout.Kind,
		})
	}

	c.JSON(http.StatusOK, res)
}
