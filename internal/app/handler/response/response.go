package response

import (
	"api/internal/app/handler/response/responsebody"
)

func Message(message string) responsebody.Message {
	return responsebody.Message{Message: message}
}
