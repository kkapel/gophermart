package handler

import (
	"gophermart/internal/config"
	db "gophermart/internal/db/connections"
	"gophermart/internal/loger"
	"gophermart/internal/services"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestHandler(t *testing.T) {
	type want struct {
		code        int
		contentType string
	}
	tests := []struct {
		name       string
		want       want
		requestURL string
		body       string
	}{
		{
			name: "Register test",
			want: want{
				code:        200,
				contentType: "application/json",
			},
			requestURL: "http://localhost:8080",
			body:       `{"login": "User1", "password": "Test1"}`,
		},
	}

	// Создаем конфиги и репозитории
	cfg := &config.Config{
		RunAddress:  "http://localhost:8080",
		DataBaseURI: "postgresql://postgres:mysecretpassword@localhost:5432/postgres", // поправить
	}

	db, _ := db.InitDB(cfg.DataBaseURI)
	// слой сервиса
	srv := services.CreateGophermartService(db.GetSqlDb())

	handler := &Handler{
		Cfg: cfg,
		Srv: srv,
	}

	//Тесты

	for _, test := range tests {
		// Общий сценарий
		// 1. Регестрируем пользователя и получаем статус 200
		t.Run(test.name, func(t *testing.T) {
			//Тестируем хэндлер RegisterUser
			requestRegisterUser := httptest.NewRequest(http.MethodPost, test.requestURL, strings.NewReader(test.body))
			postRecorder := httptest.NewRecorder()
			// тут вызвать хэндлер
			handler.RegisterUser(postRecorder, requestRegisterUser)

			resultResponse := postRecorder.Result()

			loger.Log.Info(test.name, zap.Any("response body", resultResponse.Body))
			assert.Equal(t, test.want.code, resultResponse.StatusCode)
			assert.Equal(t, test.want.contentType, resultResponse.Header.Get("Content-Type"))

		})
	}

}
