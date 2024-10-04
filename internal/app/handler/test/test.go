package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

type Case struct {
	Name    string
	Repo    func(mock sqlmock.Sqlmock)
	Request Request
	Expect  Expect
}

type Request struct {
	Body    string
	Headers map[string]string
}

type Expect struct {
	Status     int
	Body       string
	BodyFields []string
}

var InternalServerErrorResponse = Expect{
	Status: http.StatusInternalServerError,
	Body:   `{"message":"internal server error"}`,
}

func Endpoint(t *testing.T, tc Case, mock sqlmock.Sqlmock, method string, handlerPath string, requestPath string, handlers ...gin.HandlerFunc) {
	t.Run(tc.Name, func(t *testing.T) {
		if tc.Repo != nil {
			tc.Repo(mock)
		}

		gin.SetMode(gin.TestMode)
		r := gin.Default()

		r.Handle(method, handlerPath, handlers...)

		req, err := http.NewRequest(method, requestPath, strings.NewReader(tc.Request.Body))
		if err != nil {
			t.Fatal(err)
		}

		for key, value := range tc.Request.Headers {
			req.Header.Add(key, value)
		}

		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if status := w.Code; status != tc.Expect.Status {
			t.Fatalf("unexpected status code returned: got %v, want %v\n", status, tc.Expect.Status)
		}

		if tc.Expect.Body != "" && w.Body.String() != tc.Expect.Body {
			t.Fatalf("unexpected body returned: got %v, want %v\n", w.Body.String(), tc.Expect.Body)
		}

		if w.Body.String() == "" && len(tc.Expect.BodyFields) > 0 {
			t.Fatal("expected some body fields, got empty body")
		} else if len(w.Body.String()) != 0 {

			var body map[string]any
			err = json.Unmarshal(w.Body.Bytes(), &body)
			if err != nil {
				t.Fatalf("can't unmarshall response body: %v\n", err)
			}

			for _, field := range tc.Expect.BodyFields {
				value, exists := body[field]
				if !exists {
					t.Fatalf("expected body field not found: %v\n", field)
				}

				v := reflect.ValueOf(value)
				if !v.IsValid() || v.IsZero() {
					t.Fatalf("expected body field is empty: %v\n", field)
				}
			}
		}
	})
}
