package services

import (
	"context"
	"database/sql"
	"errors"
	db "gophermart/internal/db"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/jackc/pgx/v5/pgconn"
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

var ErrUniqueLogin = errors.New("Login already exists")
var ErrPasswordIncorrect = errors.New("The password is incorrect")

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
		// Проверяем отдельно ошибку дубля для логина
		var pgErr *pgconn.PgError
		if errors.As(err, pgErr) {
			if pgErr.Code == "23505" { // unique constraint violation
				return "", ErrUniqueLogin
			}
		}
		return "", err
	}

	// Формирование токена
	token, err := generateToken(user)

	return token, err

}

// Функция проверки пользователя
func (s *GophermartService) AuthUser(ctx context.Context, login string, password string) (bool, error) {

	//Получаем хэш-пароль из бд
	hashPassword, err := s.queries.GetPassword(ctx, login)
	if err != nil {
		return false, err
	}

	// Проверяем хэш
	userAuth, err := CheckPassword(password, hashPassword)

	if err != nil {
		return false, err
	}

	if userAuth {
		return true, nil
	} else {
		return false, ErrPasswordIncorrect
	}

	return false, nil

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
