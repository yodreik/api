package response

import (
	"api/internal/app/handler/response/responsebody"
)

func Err(message string) responsebody.Error {
	return responsebody.Error{Message: message}
}
