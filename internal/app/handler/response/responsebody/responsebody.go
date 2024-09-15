package responsebody

type Message struct {
	Message string `json:"message"`
}

type Account struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	IsPrivate   bool   `json:"is_private"`
	IsConfirmed bool   `json:"is_confirmed"`
}

type Profile struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	DisplayName  string    `json:"display_name"`
	IsPrivate    bool      `json:"is_private"`
	WeekActivity []Workout `json:"week_activity"`
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
