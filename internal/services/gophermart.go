package services

import (
	"context"
	"database/sql"
	db "gophermart/internal/db"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type GophermartService struct {
	queries *db.Queries
}

type Claims struct {
	jwt.RegisteredClaims
	UserID int32
}

const TokenExp = time.Hour * 3
const SecretKey = "testKey1"

func CreateGophermartService(conn *sql.DB) *GophermartService {
	return &GophermartService{
		queries: db.New(conn),
	}

}

func (s *GophermartService) RegisterUser(ctx context.Context, login string, password string) (string, error) {

	// формирование хеша для пароля
	hashPassword, err := HashPassword(password)
	if err != nil {
		return "", err
	}

	// Вызов БД
	user, err := s.queries.SaveUser(ctx, db.SaveUserParams{
		Login:    login,
		PassHash: hashPassword,
	})

	if err != nil {
		return "", err
	}

	// Формирование токена
	token, err := generateToken(user)

	return token, err

}

// Функция генерации токена
func generateToken(userID int32) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// когда создан токен
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExp)),
		},
		UserID: userID,
	})

	tokenString, err := token.SignedString([]byte(SecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
