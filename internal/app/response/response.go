package response

type Error struct {
	Message string `json:"message"`
}

func Err(message string) Error {
	return Error{Message: message}
}

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}
