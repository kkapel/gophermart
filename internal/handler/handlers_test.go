package handler

import (
	"context"
	"gophermart/internal/auth"
	"gophermart/internal/config"
	db "gophermart/internal/db/connections"
	"gophermart/internal/services"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandler(t *testing.T) {
	type want struct {
		code        int
		contentType string
	}
	type err409 struct {
		code int
	}
	type err401 struct {
		code int
		body string
	}
	type order struct {
		body string
		code int
	}
	tests := []struct {
		name   string
		want   want
		body   string
		err    err409
		err401 err401
		order  order
	}{
		{
			name: "Main test",
			want: want{
				code:        200,
				contentType: "application/json",
			},
			body: `{"login": "User12345", "password": "Test123456789"}`,
			err: err409{
				code: 409,
			},
			err401: err401{
				code: 401,
				body: `{"login": "User12345FAULT", "password": "Test123456789FAULT"}`,
			},
			order: order{
				body: "12345678903",
				code: 202,
			},
		},
	}

	// Создаем конфиги и репозитории
	cfg := &config.Config{
		RunAddress:  "http://localhost:8080",
		DataBaseURI: "postgresql://postgres:admin@localhost:5432/postgres?sslmode=disable", // поправить
	}

	// Проверяем путь к миграциям
	entries, err := os.ReadDir("../../migrations")
	if err != nil {
		t.Logf("Ошибка доступа к миграциям: %v", err)
	} else {
		for _, e := range entries {
			t.Logf("Нашел файл миграции: %s", e.Name())
		}
	}

	database, err := db.InitDB(cfg.DataBaseURI, "file://../../migrations")
	if err != nil {
		t.Fatalf("Failed to connect to DB: %v", err)
	}
	// слой сервиса
	srv := services.CreateGophermartService(database.GetSqlDb())

	handler := &Handler{
		Cfg: cfg,
		Srv: srv,
	}

	//Тесты
	for _, test := range tests {
		// Общий сценарий
		// 1. Регестрируем пользователя и получаем статус 200
		// 2. Делаем повторный запрос на регистрацию с такими же данными и получаем ошибку 409.
		// 3. Проверяем хэндлер авторизации
		// 4. Загружаем заказ

		t.Run(test.name, func(t *testing.T) {

			// чистим таблицы
			_, err := database.GetSqlDb().Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE")
			if err != nil {
				t.Fatalf("failed to truncate table: %v", err)
			}
			_, err = database.GetSqlDb().Exec("TRUNCATE TABLE orders RESTART IDENTITY CASCADE")
			if err != nil {
				t.Fatalf("failed to truncate table: %v", err)
			}

			//Тестируем хэндлер RegisterUser
			requestRegisterUser := httptest.NewRequest(http.MethodPost, handler.Cfg.RunAddress, strings.NewReader(test.body))
			postRecorder := httptest.NewRecorder()
			// тут вызвать хэндлер
			handler.RegisterUser(postRecorder, requestRegisterUser)

			resultResponse := postRecorder.Result()
			defer resultResponse.Body.Close()

			//t.Logf("Response Body: %s", postRecorder.Body.String()) // Выводим тело ответа
			assert.Equal(t, test.want.code, resultResponse.StatusCode)
			// Проверяем что заполнился хедер Authorization
			assert.NotEmpty(t, resultResponse.Header.Get("Authorization"), "Authorization should not be empty")

			// Пробуем второй раз внести логин и пароль
			// Должны получить ошибку 409
			requestRegisterUser409 := httptest.NewRequest(http.MethodPost, handler.Cfg.RunAddress, strings.NewReader(test.body))
			postRecorder409 := httptest.NewRecorder()
			handler.RegisterUser(postRecorder409, requestRegisterUser409)

			resultResponse409 := postRecorder409.Result()
			defer resultResponse409.Body.Close()
			assert.Equal(t, test.err.code, resultResponse409.StatusCode)

			// Далее проверяем хэндлер POST /api/user/login
			requestAuthUser := httptest.NewRequest(http.MethodPost, handler.Cfg.RunAddress, strings.NewReader(test.body))
			postRecorderAuth := httptest.NewRecorder()

			handler.UserAuth(postRecorderAuth, requestAuthUser)
			resultResponseAuth := postRecorderAuth.Result()
			defer resultResponseAuth.Body.Close()

			assert.Equal(t, test.want.code, resultResponseAuth.StatusCode)
			assert.NotEmpty(t, resultResponseAuth.Header.Get("Authorization"), "Authorization should not be empty")
			//token := resultResponseAuth.Header.Get("Authorization")
			// Отправим неверный логин и пароль
			requestAuthUser401 := httptest.NewRequest(http.MethodPost, handler.Cfg.RunAddress, strings.NewReader(test.err401.body))
			postRecorderAuth401 := httptest.NewRecorder()
			handler.UserAuth(postRecorderAuth401, requestAuthUser401)

			resultResponseAuth401 := postRecorderAuth401.Result()
			defer resultResponseAuth401.Body.Close()
			assert.Equal(t, test.err401.code, resultResponseAuth401.StatusCode)

			// Далее проверяем хэндлер POST /api/user/orders
			requestOrder := httptest.NewRequest(http.MethodPost, handler.Cfg.RunAddress, strings.NewReader(test.order.body))
			//requestOrder.Header.Set("Authorization", token)
			ctx := context.WithValue(requestOrder.Context(), auth.UserIDKey, int32(1))
			requestOrder = requestOrder.WithContext(ctx)
			postRecorderOrder := httptest.NewRecorder()

			handler.SaveOrder(postRecorderOrder, requestOrder)
			resultOrder := postRecorderOrder.Result()
			t.Logf("Response Body: %s", postRecorderOrder.Body.String()) // Выводим тело ответа
			defer resultOrder.Body.Close()

			assert.Equal(t, test.order.code, resultOrder.StatusCode)

		})
	}

}
