package services

import (
	"context"
	"database/sql"
	"errors"
	"gophermart/internal/accrual"
	db "gophermart/internal/db"
	"gophermart/internal/loger"
	"gophermart/internal/repository"
	"iter"
	"log/slog"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
)

type GophermartService struct {
	accrual    *accrual.Accrual
	repository *repository.Repository
}

type Claims struct {
	jwt.RegisteredClaims
	UserID int32
}

type Order struct {
	Number     string          `json:"number"`
	Status     string          `json:"status"`
	Accrual    decimal.Decimal `json:"accrual,omitempty"`
	UploadedAt time.Time       `json:"uploaded_at"`
}

type UserBalance struct {
	Current   decimal.Decimal `json:"current"`
	Withdrawn decimal.Decimal `json:"withdrawn"`
}

type Withdrawals struct {
	Order       string          `json:"order"`
	Sum         decimal.Decimal `json:"sum"`
	ProccesedAt time.Time       `json:"processed_at"`
}

const TokenExp = time.Hour * 3
const SecretKey = "testKey1"
const period = 200
const batchSize = 100

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
	ErrNotEnoughAccrualPoints     = errors.New("Not enough accrual points")
	ErrNotExistsWithdrawals       = errors.New("Withdrawals not exists")
)

func CreateGophermartService(conn *sql.DB, accrual *accrual.Accrual) *GophermartService {

	repo := repository.NewRepository(db.New(conn))
	gophermart := &GophermartService{
		accrual:    accrual,
		repository: repo,
	}
	return gophermart
}

func (s *GophermartService) RegisterUser(ctx context.Context, login string, password string) (string, error) {

	// формирование хеша для пароля
	hashPassword, err := HashPassword(password)
	if err != nil {
		return "", err
	}

	// Вызов БД
	user, err := s.repository.SaveUser(ctx, db.SaveUserParams{
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
	rows, err := s.repository.GetPassword(ctx, login)
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

	loger.Log.Info("Save order service", slog.String("orderNumber", orderNumber))
	loger.Log.Info("Save order service", slog.Any("userID", userID))
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
	id, err := s.repository.GetUserIDByOrder(ctx, orderNumber)

	if err == nil {
		if id == userID {
			return ErrOrderByUserLoaded
		}
		return ErrOrderLoaded //Заказ есть, но загружен другим пользователем
	}

	if err != sql.ErrNoRows {
		return err
	}

	// Сохраняем номер заказа в БД
	s.repository.SaveOrder(ctx, db.SaveOrderParams{
		OrderNumber: orderNumber,
		Status:      "NEW", // Новый заказ
		UploadedAt:  time.Now(),
		UserID:      userID,
	})

	return nil
}

func (s *GophermartService) GetOrders(ctx context.Context, userID int32) (iter.Seq[Order], error) {

	// Вызываем модуль БД
	ordersDB, err := s.repository.GetOrdersByUsers(ctx, userID)

	if err != nil {
		return nil, err
	}

	// БД вернула пустой SELECT
	if len(ordersDB) == 0 {
		return nil, ErrOrderListIsEmpty
	}

	seq := func(yield func(Order) bool) {
		for _, order := range ordersDB {
			outOrder := Order{
				Number:     order.OrderNumber,
				Status:     order.Status,
				Accrual:    ToDecimalFromDB(order.Accrual.Int64),
				UploadedAt: order.UploadedAt,
			}

			if !yield(outOrder) {
				return
			}
		}

	}

	return seq, nil
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
		loger.Log.Info("gophermart.go", slog.String("Func CheckToken", "Token is not valid"))

		return 0, ErrTokenIsNotValid
	}

	//Если userID не заполнен
	if claims.UserID < 1 {
		loger.Log.Info("gophermart.go", slog.String("Func CheckToken", "Token is not valid. UserID is empty"))
		return 0, ErrUserIDNotFound
	}

	loger.Log.Info("gophermart.go", slog.String("Func CheckToken", "Token is valid"))
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

func (s *GophermartService) GetOrdersForAccrual() {
	ticker := time.NewTicker(period * time.Millisecond)
	defer ticker.Stop()

	const numWorkers = 5
	ordersChan := make(chan accrual.InputAccrualType, batchSize)
	resultChan := make(chan accrual.OrderWithAccrual, numWorkers)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запускаем go-рутины
	//var wg sync.WaitGroup
	eg, ctx := errgroup.WithContext(ctx)
	for i := 0; i < 5; i++ {
		eg.Go(func() error {
			return accrual.SaveOrder(ctx, ordersChan, resultChan, s.accrual.URL)
		})
	}

	go func() {
		if err := eg.Wait(); err != nil {
			loger.Log.Error("Error channel in GetOrdersForAccrual func", slog.String("error", err.Error()))
			cancel()
		}
		close(resultChan)
	}()

	for {
		select {
		case <-ticker.C:
			// Получаем новые заказы для отправки в accrual
			args := db.GetOrdersForAccrualParams{
				Status: "NEW",
				Limit:  batchSize,
			}
			orders, err := s.repository.GetOrdersForAccrual(ctx, args)
			if err != nil {
				// В случае ошибки просто логируем
				loger.Log.Error(err.Error())
			} else {

				for _, order := range orders {
					select {
					case ordersChan <- accrual.InputAccrualType{
						Order:        order,
						AccrualMutex: s.accrual.AccrualMutex,
					}:
					case <-ctx.Done():
						return
					}
				}
			}
		case resultOrder := <-resultChan:
			// Пишем результат в БД
			update := db.UpdateOrderStatusParams{
				Status: resultOrder.Status,
				Accrual: sql.NullInt64{
					Int64: FromDecimalToDB(resultOrder.Accrual),
					Valid: FromDecimalToDB(resultOrder.Accrual) != 0}, // Если accrual = 0, то пишем null в бд
				OrderNumber: resultOrder.Order,
			}
			row, errorUpdate := s.repository.UpdateOrderStatus(ctx, update)
			if errorUpdate != nil {
				loger.Log.Error("GetOrdersForAccrual func", slog.String("Update db error", errorUpdate.Error()))
			}
			if row == 0 {
				loger.Log.Error("GetOrdersForAccrual func", slog.String("Update db error", "Update statement return 0 rows"))
			}
		case <-ctx.Done():
			close(ordersChan)
			loger.Log.Info("GetOrdersForAccrual func", slog.String("ctx Done", ""))
			return
		}

	}
}

func (s *GophermartService) GetUserBalance(ctx context.Context, userID int32) (*UserBalance, error) {
	// Вызов БД
	balance, err := s.repository.GetBalance(ctx, userID)

	if err != nil {
		loger.Log.Error("GetUserBalance func", slog.String("db GetBalance error", err.Error()))
		return nil, err
	}

	withdrawn, err := s.repository.GetWithdraws(ctx, userID)

	if err != nil {
		loger.Log.Error("GetUserBalance func", slog.String("db GetWithdraws error", err.Error()))
		return nil, err
	}

	current := balance - withdrawn

	return &UserBalance{
		Current:   ToDecimalFromDB(current),
		Withdrawn: ToDecimalFromDB(withdrawn),
	}, nil
}

func (s *GophermartService) Withdraw(ctx context.Context, userID int32, orderNumber string, sum decimal.Decimal) error {
	// Вызов функции на стороне БД
	ok, err := s.repository.WithdrawDB(ctx,
		db.WithdrawDBParams{
			InputOrderNumber: orderNumber,
			InputWithdraw:    FromDecimalToDB(sum),
			InputUserID:      userID,
		})
	if err != nil {
		loger.Log.Error("Withdraw func", slog.String("WithdrawDB error", err.Error()))
		return err
	}

	if !ok {
		// Не удалось списать
		// Не хватает баллов
		return ErrNotEnoughAccrualPoints
	}
	return nil
}

func (s *GophermartService) GetWithdrawals(ctx context.Context, userID int32) (*[]Withdrawals, error) {
	withdrawals, err := s.repository.GetAllWithdrawals(ctx, userID)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotExistsWithdrawals
		}
		loger.Log.Error("GetWithdrawals", slog.String("GetWithdrawals db error", err.Error()))
		return nil, err
	}

	result := make([]Withdrawals, len(withdrawals))
	for i, v := range withdrawals {
		result[i].Order = v.OrderNumber
		result[i].ProccesedAt = v.ProcessedAt.Time
		result[i].Sum = ToDecimalFromDB(v.Withdraw.Int64)
	}

	return &result, nil

}

func ToDecimalFromDB(inputNumber int64) decimal.Decimal {
	return decimal.New(inputNumber, -2)
}

func FromDecimalToDB(inputNumber decimal.Decimal) int64 {
	return inputNumber.Mul(decimal.NewFromInt(100)).IntPart()
}
