package handler

import (
	"encoding/json"
	"gophermart/internal/auth"
	"gophermart/internal/config"
	"gophermart/internal/loger"
	"gophermart/internal/services"
	"io"
	"log/slog"
	"net/http"
	"slices"

	"github.com/go-playground/validator/v10"
	"github.com/shopspring/decimal"
)

type Handler struct {
	Cfg *config.Config
	Srv *services.GophermartService
}

type UserRegister struct {
	Login    string `json:"login" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type WithdrawRequest struct {
	Order string          `json:"order"`
	Sum   decimal.Decimal `json:"sum"`
}

func (h *Handler) RegisterUser(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		loger.Log.Info("handlers.go", slog.String("Function RegisterUser", "Starts function"))
		body, err := io.ReadAll(req.Body)

		if err != nil {
			loger.Log.Error("handlers.go", slog.String("Function RegisterUser", "Can not read request body"))
			http.Error(res, "Cannot read request body", http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		// Парсим JSON
		var userRegister UserRegister
		if err := json.Unmarshal(body, &userRegister); err != nil {
			// Ошибка 400
			loger.Log.Error("handlers.go", slog.String("Function RegisterUser", "Can not unmarshal request body"))
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		if err := validateUserRegister(&userRegister); err != nil {
			// Ошибка 400
			loger.Log.Error("handlers.go", slog.String("Function RegisterUser", "Request validate error"))
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		// Вызывам дальнейшую обработку в слое сервиса
		token, err := h.Srv.RegisterUser(req.Context(), userRegister.Login, userRegister.Password)

		// отдельно нужно обработать ошибку 409 - Логин уже занят
		if err == services.ErrUniqueLogin {
			http.Error(res, err.Error(), http.StatusConflict)
			return
		} else if err != nil {
			loger.Log.Error("handlers.go", slog.String("Function RegisterUser", err.Error()))
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}

		// Заполняем хедер "Authorization"
		res.Header().Set("Authorization", "Bearer "+token)
		res.WriteHeader(http.StatusOK)

	default:
		errorResponse(res)
	}

}

// Аутентификация пользователя
func (h *Handler) UserAuth(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		loger.Log.Info("handlers.go", slog.String("Function UserAuth", "Starts function"))
		body, err := io.ReadAll(req.Body)

		if err != nil {
			loger.Log.Error("handlers.go", slog.String("Function UserAuth", "Can not read request body"))
			http.Error(res, "Cannot read request body", http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		var userRegister UserRegister
		if err := json.Unmarshal(body, &userRegister); err != nil {
			// Ошибка 400
			loger.Log.Error("handlers.go", slog.String("Function UserAuth", "Can not unmarshal request body"))
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		if err := validateUserRegister(&userRegister); err != nil {
			// Ошибка 400
			loger.Log.Error("handlers.go", slog.String("Function UserAuth", "Request validate error"))
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		auth, token, err := h.Srv.AuthUser(req.Context(), userRegister.Login, userRegister.Password)

		//Ошибка логин/пароль
		if err == services.ErrPasswordIncorrect {
			// Status code 401
			http.Error(res, err.Error(), http.StatusUnauthorized)
			return
		} else if err != nil {
			loger.Log.Error("handlers.go", slog.String("Function UserAuth", err.Error()))
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}

		if auth {
			// Заполняем хедер "Authorization"
			res.Header().Set("Authorization", "Bearer "+token)
			res.WriteHeader(http.StatusOK)
			return
		}

	default:
		errorResponse(res)
	}

}

// Функция сохранения номера заказа
func (h *Handler) SaveOrder(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		loger.Log.Info("handlers.go", slog.String("Function SaveOrder", "Starts function"))
		body, err := io.ReadAll(req.Body)

		if err != nil {
			loger.Log.Error("handlers.go", slog.String("Function SaveOrder", "Can not read request body"))
			http.Error(res, "Cannot read request body", http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		orderNumber := string(body)

		err = h.Srv.SaveOrder(req.Context(), orderNumber, req.Context().Value(auth.UserIDKey).(int32))
		if err == services.ErrOrderByUserLoaded {
			// Статус
			res.WriteHeader(http.StatusOK)
			return
		} else if err == services.ErrOrderLoaded {
			http.Error(res, err.Error(), http.StatusConflict)
			return
		} else if err == services.ErrIncorrectOrderNumberFormat {
			//402
			http.Error(res, err.Error(), http.StatusUnprocessableEntity)
			return
		} else if err != nil { // Все остальные ошибки
			loger.Log.Error("handlers.go", slog.String("Function SaveOrder", err.Error()))
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		// Статус 202
		res.WriteHeader(http.StatusAccepted)

	default:
		errorResponse(res)
	}
}

func (h *Handler) GetOrders(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		loger.Log.Info("handlers.go", slog.String("Function GetOrders", "Starts function"))

		// Вызываем метод получения списка заказов
		seq, err := h.Srv.GetOrders(req.Context(), req.Context().Value(auth.UserIDKey).(int32))

		if err == services.ErrOrderListIsEmpty {
			loger.Log.Error("handlers.go", slog.String("Function GetOrders", err.Error()))
			res.WriteHeader(http.StatusNoContent)
			return
		}

		if err != nil {
			// 500
			loger.Log.Error("handlers.go", slog.String("Function GetOrders", err.Error()))

			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		orders := slices.Collect(seq)

		resp, err := json.Marshal(orders)

		if err != nil {
			loger.Log.Error("handlers.go", slog.String("Function GetOrders", err.Error()))

			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		res.Header().Set("content-type", "application/json")
		res.WriteHeader(http.StatusOK)
		loger.Log.Info("handlers.go", slog.String("Function GetOrders result", string(resp)))
		res.Write(resp)

	default:
		errorResponse(res)
	}

}

// Получение баланса
func (h *Handler) GetUserBalance(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		loger.Log.Info("handlers.go", slog.String("Function GetUserBalance", "Starts function"))

		userBalance, err := h.Srv.GetUserBalance(req.Context(), req.Context().Value(auth.UserIDKey).(int32))

		if err != nil {
			loger.Log.Error("handlers.go", slog.String("Function GetUserBalance", err.Error()))
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		resp, err := json.Marshal(userBalance)

		if err != nil {
			loger.Log.Error("handlers.go", slog.String("Function GetUserBalance", err.Error()))
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		res.Header().Set("content-type", "application/json")
		res.WriteHeader(http.StatusOK)
		loger.Log.Info("handlers.go", slog.String("Function GetUserBalance result", string(resp)))
		res.Write(resp)

	default:
		errorResponse(res)
	}
}

// Запрос на списание средств
func (h *Handler) Withdraw(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		loger.Log.Info("handlers.go", slog.String("Function Withdraw", "Starts function"))

		body, err := io.ReadAll(req.Body)

		if err != nil {
			loger.Log.Error("handlers.go", slog.String("Function Withdraw", "Can not read request body"))
			http.Error(res, "Cannot read request body", http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		var withdraw WithdrawRequest
		if err := json.Unmarshal(body, &withdraw); err != nil {
			// Ошибка 500
			loger.Log.Error("handlers.go", slog.String("Function Withdraw", "Can not unmarshal request body"))
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		// Вызов слоя сервиса
		err = h.Srv.Withdraw(req.Context(), req.Context().Value(auth.UserIDKey).(int32), withdraw.Order, withdraw.Sum)

		if err != nil {
			if err == services.ErrNotEnoughAccrualPoints {
				// 402
				loger.Log.Error("handlers.go", slog.String("Function Withdraw", err.Error()))
				http.Error(res, err.Error(), http.StatusPaymentRequired)
				return
			} else {
				loger.Log.Error("handlers.go", slog.String("Function Withdraw", err.Error()))
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// 200
		res.WriteHeader(http.StatusOK)

	default:
		errorResponse(res)
	}
}

func (h *Handler) GetWithdrawals(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		// Вызываем слой сервиса
		result, err := h.Srv.GetWithdrawals(req.Context(), req.Context().Value(auth.UserIDKey).(int32))

		if err != nil {
			if err == services.ErrNotExistsWithdrawals {
				res.WriteHeader(http.StatusNoContent)
				return
			}
			loger.Log.Error("handlers.go", slog.String("Function GetWithdrawals", err.Error()))
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		resp, err := json.Marshal(result)

		if err != nil {
			loger.Log.Error("handlers.go", slog.String("Function GetWithdrawals", err.Error()))
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		res.Header().Set("content-type", "application/json")
		res.WriteHeader(http.StatusOK)
		res.Write(resp)
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
