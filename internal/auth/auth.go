package auth

import (
	"context"
	"gophermart/internal/services"
	"net/http"
	"strings"
)

type contextKey string

const UserIDKey contextKey = "userID"

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Получаем хедер
		auth := r.Header.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			// Ошибка 401
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token := strings.Trim(auth, "Bearer ")

		userID, err := services.CheckToken(token) // Вызываем метод проверки токена

		if err != nil {

		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
