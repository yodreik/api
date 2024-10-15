package handler

import (
	"api/internal/app/handler/response/responsebody"
	"api/internal/app/handler/test"
	"api/internal/config"
	mockmailer "api/internal/mailer/mock"
	"api/internal/repository"
	mocktoken "api/internal/token/mock"
	"net/http"
	"testing"
)

func TestUserIdentity(t *testing.T) {
	c := config.Config{}
	repo := repository.Repository{}
	handler := New(&c, &repo, mockmailer.New(), mocktoken.New(c.Token))

	tests := []test.Case{
		{
			Name: "empty header",

			Expect: test.Expect{
				Status: http.StatusUnauthorized,
				Body: responsebody.Message{
					Message: "empty authorization header",
				},
			},
		},
		{
			Name: "invalid token type",

			Request: test.Request{
				Headers: map[string]string{
					"Authorization": "Bot <token>",
				},
			},

			Expect: test.Expect{
				Status: http.StatusUnauthorized,
				Body: responsebody.Message{
					Message: "invalid authorization token type",
				},
			},
		},
		{
			Name: "incorrect token format",

			Request: test.Request{
				Headers: map[string]string{
					"Authorization": "Bearer some.incorrect.jwonwebtoken",
				},
			},

			Expect: test.Expect{
				Status: http.StatusUnauthorized,
				Body: responsebody.Message{
					Message: "invalid authorization token",
				},
			},
		},
	}

	for _, tc := range tests {
		test.Endpoint(t, tc, nil, http.MethodGet, "/api/me", "/api/me", handler.UserIdentity)
	}
}
