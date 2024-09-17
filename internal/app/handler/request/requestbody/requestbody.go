package requestbody

type CreateAccount struct {
	Email    string `json:"email" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type CreateSession struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type CreateWorkout struct {
	Date     string `json:"date" binding:"required"`
	Duration int    `json:"duration" binding:"required"`
	Kind     string `json:"kind" binding:"required"`
}

type ResetPassword struct {
	Email string `json:"email" binding:"required"`
}

type UpdatePassword struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type ConfirmAccount struct {
	Token string `json:"token" binding:"required"`
}
