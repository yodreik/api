package response

import (
	responsebody "api/internal/app/handler/response/body"
)

func Err(message string) responsebody.Error {
	return responsebody.Error{Message: message}
}
