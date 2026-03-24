package handler

import (
	"encoding/json"
	"gophermart/internal/config"
	"gophermart/internal/loger"
	"gophermart/internal/services"
	"io"
	"net/http"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type Handler struct {
	Cfg *config.Config
	Srv *services.GophermartService
}

type UserRegister struct {
	Login    string `json:"login" validate:"required, min=6"`
	Password string `json:"password" validate:"required, min=8"`
}

func (h *Handler) RegisterUser(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		loger.Log.Info("handlers.go", zap.String("Function RegisterUser", "Starts function"))
		body, err := io.ReadAll(req.Body)

		if err != nil {
			loger.Log.Error("handlers.go", zap.String("Function RegisterUser", "Can not read request body"))
			http.Error(res, "Cannot read request body", http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		// Парсим JSON
		var userRegister UserRegister
		if err := json.Unmarshal(body, &userRegister); err != nil {
			// Ошибка 400
			loger.Log.Error("handlers.go", zap.String("Function RegisterUser", "Can not unmarshal request body"))
			http.Error(res, err.Error(), http.StatusBadRequest)
		}

		if err := validateUserRegister(&userRegister); err != nil {
			// Ошибка 400
			loger.Log.Error("handlers.go", zap.String("Function RegisterUser", "Request validate error"))
			http.Error(res, err.Error(), http.StatusBadRequest)
		}

		// Вызывам дальнейшую обработку в слое сервиса
		token, err := h.Srv.RegisterUser(req.Context(), userRegister.Login, userRegister.Password)

		// отдельно нужно обработать ошибку 409 - Логин уже занят

		// Заполняем хедер "Authorization"
		res.Header().Set("Authorization", "Bearer "+token)
		res.WriteHeader(http.StatusOK)

	default:
		errorResponse(res)
	}

}

// Функция валидации входной структуры
func validateUserRegister(userRegister *UserRegister) error {
	validator := validator.New()
	if err := validator.Struct(userRegister); err != nil {
		return err
	}

	return nil
}

func errorResponse(res http.ResponseWriter) {
	res.WriteHeader(http.StatusBadRequest)
}
