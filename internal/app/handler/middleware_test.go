package handler

import (
	"api/internal/config"
	"api/internal/repository"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUserIdentity(t *testing.T) {
	c := config.Config{}
	repo := repository.Repository{}

	tt := []table{
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

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.Default()

			handler := New(&c, &repo)

			r.GET("/api/me", handler.UserIdentity, handler.Me)

			req, err := http.NewRequest(http.MethodGet, "/api/me", strings.NewReader(tc.request.body))
			if err != nil {
				t.Fatal(err)
			}

			for key, value := range tc.request.headers {
				req.Header.Add(key, value)
			}

			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if status := w.Code; status != tc.expect.status {
				t.Fatalf("handler returned wrong status code: got %v, want %v\n", status, tc.expect.status)
			}

			if w.Body.String() != tc.expect.body {
				t.Fatalf("handler returned unexpected body: got %v, want %v\n", w.Body.String(), tc.expect.body)
			}
		})
	}
}
