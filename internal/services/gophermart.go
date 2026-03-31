package services

import (
	"context"
	"database/sql"
	"errors"
	db "gophermart/internal/db"
	"gophermart/internal/loger"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

type GophermartService struct {
	queries *db.Queries
}

type Claims struct {
	jwt.RegisteredClaims
	UserID int32
}

type Order struct {
	number      string    `json:"number"`
	status      string    `json:"status"`
	accrual     int32     `json:"accrual"`
	uploaded_at time.Time `json:"uploaded_at"`
}

const TokenExp = time.Hour * 3
const SecretKey = "testKey1"

var (
	ErrUniqueLogin                = errors.New("Login already exists")
	ErrPasswordIncorrect          = errors.New("The password is incorrect")
	ErrIncorrectOrderNumberFormat = errors.New("Order number is incorrect")
	TokenParsingError             = errors.New("TokenParsingError")
	ErrTokenIsNotValid            = errors.New("TokenIsNotValid")
	ErrUserIDNotFound             = errors.New("UserIDNotFound")
	ErrOrderByUserLoaded          = errors.New("The order has been loaded by this user")
	ErrOrderLoaded                = errors.New("The order has been loaded by other user")
	ErrOrderListIsEmpty           = errors.New("Order list is empty")
)

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
func (s *GophermartService) AuthUser(ctx context.Context, login string, password string) (bool, string, error) {

	//Получаем хэш-пароль из бд
	rows, err := s.queries.GetPassword(ctx, login)
	if err != nil {
		return false, "", err
	}

	// Если не нашли пароль в бд
	if len(rows) == 0 {
		return false, "", ErrPasswordIncorrect
	}

	// Проверяем хэш
	userAuth, err := CheckPassword(password, rows[0].PassHash)

	if err != nil {
		return false, "", err
	}

	if !userAuth {
		return false, "", ErrPasswordIncorrect
	}

	token, err := generateToken(rows[0].ID)

	return true, token, err
}

// Функция сохранения номера заказа
func (s *GophermartService) SaveOrder(ctx context.Context, orderNumber string, userID int32) error {
	// Логирование для автотестов

	loger.Log.Info("Save order service", zap.String("orderNumber", orderNumber))
	loger.Log.Info("Save order service", zap.Int32("userID", userID))
	// Делаем проверку, что пришло число
	_, err := strconv.Atoi(orderNumber)
	if err != nil {
		return ErrPasswordIncorrect
	}

	// Проверяем на алгоритм Луна
	if !CheckLuhnAlgorithm(orderNumber) {
		return ErrIncorrectOrderNumberFormat
	}

	// Проверяем что заказ еще не добавлен
	id, err := s.queries.GetUserIDByOrder(ctx, orderNumber)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if id == userID {
		return ErrOrderByUserLoaded
	} else if id != 0 { // Заказ есть, но загружен другим пользователем
		return ErrOrderLoaded
	}

	// Сохраняем номер заказа в БД
	s.queries.SaveOrder(ctx, db.SaveOrderParams{
		OrderNumber: orderNumber,
		Status:      "NEW", // Новый заказ
		UploadedAt:  time.Now(),
		UserID:      userID,
	})

	return nil
}

func (s *GophermartService) GetOrders(ctx context.Context, userID int32) (*[]Order, error) {

	// Вызываем модуль БД
	ordersDB, err := s.queries.GetOrdersByUsers(ctx, userID)

	if err != nil {
		return nil, err
	}

	// БД вернула пустой SELECT
	if ordersDB == nil {
		return nil, ErrOrderListIsEmpty
	}

	var orders []Order

	for i, order := range ordersDB {
		orders[i] = Order{
			number:      order.OrderNumber,
			status:      order.Status,
			accrual:     order.Accrual.Int32,
			uploaded_at: order.UploadedAt,
		}
	}
	return &orders, nil
}

// Функция проверки токена
func CheckToken(tokenString string) (int32, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			return []byte(SecretKey), nil
		})
	if err != nil {
		return 0, TokenParsingError
	}

	if !token.Valid {
		loger.Log.Info("gophermart.go", zap.String("Func CheckToken", "Token is not valid"))

		return 0, ErrTokenIsNotValid
	}

	//Если userID не заполнен
	if claims.UserID < 1 {
		loger.Log.Info("gophermart.go", zap.String("Func CheckToken", "Token is not valid. UserID is empty"))
		return 0, ErrUserIDNotFound
	}

	loger.Log.Info("gophermart.go", zap.String("Func CheckToken", "Token is valid"))
	return claims.UserID, nil
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
