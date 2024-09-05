package handler

import (
	"api/internal/config"
	mockmailer "api/internal/mailer/mock"
	"api/internal/repository"
	"net/http"
	"testing"
)

func TestUserIdentity(t *testing.T) {
	c := config.Config{}
	repo := repository.Repository{}
	handler := New(&c, &repo, mockmailer.New())

	tests := []table{
		{
			name: "empty header",

			request: request{
				headers: map[string]string{
					"Authorization": "", // it can be totally removed, keep it just for the sake of the
				},
			},

			expect: expect{
				status: http.StatusUnauthorized,
				body:   `{"message":"empty authorization header"}`,
			},
		},
		{
			name: "invalid token type",

			request: request{
				headers: map[string]string{
					"Authorization": "Bot <token>",
				},
			},

			expect: expect{
				status: http.StatusUnauthorized,
				body:   `{"message":"invalid authorization token type"}`,
			},
		},
		{
			name: "incorrect token format",

			request: request{
				headers: map[string]string{
					"Authorization": "Bearer some.incorrect.jwonwebtoken",
				},
			},

			expect: expect{
				status: http.StatusUnauthorized,
				body:   `{"message":"invalid authorization token"}`,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, TemplateTestHandler(tc, nil, http.MethodGet, "/api/me", handler.UserIdentity))
	}
}
