package requestbody

type CreateAccount struct {
	Email    string `json:"email" binding:"required,max=254"`
	Username string `json:"username" binding:"required,min=5,max=32"`
	Password string `json:"password" binding:"required,min=8,max=64"`
}

type CreateSession struct {
	Login    string `json:"login" binding:"required,max=254"`
	Password string `json:"password" binding:"required"`
}

type UpdateAccount struct {
	Email       *string `json:"email" binding:"omitempty,max=254"`
	Username    *string `json:"username" binding:"omitempty,min=5,max=32"`
	DisplayName *string `json:"display_name" binding:"omitempty,max=50"`
	Password    *string `json:"password" binding:"omitempty,min=8,max=64"`
	IsPrivate   *bool   `json:"is_private" binding:"omitempty"`
}

type CreateWorkout struct {
	Date     string `json:"date" binding:"required"`
	Duration int    `json:"duration" binding:"required"`
	Kind     string `json:"kind" binding:"required"`
}

type ResetPassword struct {
	Email string `json:"email" binding:"required,max=254"`
}

type UpdatePassword struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required,min=8,max=64"`
}

type ConfirmAccount struct {
	Token string `json:"token" binding:"required"`
}
