package services

import (
	"context"
	"database/sql"
	"errors"
	db "gophermart/internal/db"
	"strconv"
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
var ErrIncorrectOrderNumberFormat = errors.New("Order number is incorrect")

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
		if errors.As(err, &pgErr) {
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
}

// Функция сохранения номера заказа
func (s *GophermartService) SaveOrder(ctx context.Context, orderNumber string) error {
	// Делаем проверку, что пришло число
	_, err := strconv.Atoi(orderNumber)
	if err != nil {
		return ErrPasswordIncorrect
	}

	// Проверяем на алгоритм Луна
	if !CheckLuhnAlgorithm(orderNumber) {
		return ErrIncorrectOrderNumberFormat
	}

	// Сохраняем номер заказа в БД
	s.queries.SaveOrder(ctx, db.SaveOrderParams{
		OrderNumber: orderNumber,
		Status:      "NEW", // Новый заказ
		UploadedAt:  time.Now(),
	})

	return nil
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

// Алгоритм Луна
func CheckLuhnAlgorithm(orderNumber string) bool {
	sum := 0
	nDigits := len(orderNumber)
	parity := nDigits % 2
	for i := 0; i < nDigits; i++ {
		digit, err := strconv.Atoi(string(orderNumber[i]))
		if err != nil {
			return false
		}

		if i%2 == parity {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}
	return sum%10 == 0
}
