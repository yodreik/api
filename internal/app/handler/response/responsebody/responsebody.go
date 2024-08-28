package responsebody

type Error struct {
	Message string `json:"message"`
}

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type Token struct {
	Token string `json:"token"`
}
