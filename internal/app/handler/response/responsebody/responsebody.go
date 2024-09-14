package responsebody

type Message struct {
	Message string `json:"message"`
}

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type Workout struct {
	ID       string `json:"id"`
	Date     string `json:"date"`
	Duration int    `json:"duration"`
	Kind     string `json:"kind"`
}

type ActivityHistory struct {
	UserID   string    `json:"user_id"`
	Count    int       `json:"count"`
	Workouts []Workout `json:"workouts"`
}

type Token struct {
	Token string `json:"token"`
}
