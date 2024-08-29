package requestbody

import "time"

type Register struct {
	Email    string `json:"email" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type Login struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type CreateWorkout struct {
	Date     time.Time `json:"date"`
	Duration int       `json:"duration" binding:"required"`
	Kind     string    `json:"kind" binding:"required"`
}
