package accrual

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

type Accrual struct {
	URL          string
	AccrualMutex *AccrualMutex
}

type AccrualMutex struct {
	Mu         *sync.RWMutex
	LockedTime *time.Time
}

type AccrualResponse struct {
	Order   string          `json:"order"`
	Status  string          `json:"status"`
	Accrual decimal.Decimal `json:"accrual"`
}

type OrderWithAccrual struct {
	Order   string
	Accrual decimal.Decimal
	Status  string
}

type InputAccrualType struct {
	Order        string
	AccrualMutex *AccrualMutex
}

func NewAccrual(url string) *Accrual {
	return &Accrual{
		URL: url,
		AccrualMutex: &AccrualMutex{
			Mu:         &sync.RWMutex{},
			LockedTime: &time.Time{},
		},
	}
}

// Функция отправки запроса в accrual
func SaveOrder(ctx context.Context, orders chan InputAccrualType, results chan<- OrderWithAccrual, urlAccrualBase string) error {

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case input, ok := <-orders:
			if !ok {
				return nil
			}
			accrualMutex := input.AccrualMutex
			order := input.Order
			mu := accrualMutex.Mu
			timeNow := time.Now()

			mu.Lock()
			timeSleep := *accrualMutex.LockedTime
			mu.Unlock()

			// Задержка перед запросом
			if timeSleep.After(timeNow) {
				time.Sleep(timeSleep.Sub(timeNow))
			}

			accrualURL, err := url.JoinPath(urlAccrualBase, "/api/orders/", order)

			if err != nil {
				return err
			}

			resp, err := http.Get(accrualURL)
			if err != nil {
				return err
			}

			// читаем статус
			switch resp.StatusCode {
			case http.StatusOK:
				var accrualResponse AccrualResponse
				err = json.NewDecoder(resp.Body).Decode(&accrualResponse)
				if err != nil {
					return err
				}
				switch accrualResponse.Status {
				case "PROCESSED":
					//если пришел успешный ответ и баллы
					// Пишем в канал результатов
					results <- OrderWithAccrual{
						Order:   order,
						Accrual: accrualResponse.Accrual,
						Status:  "PROCESSED",
					}
				case "REGISTERED":
					// Заказ зарегистрирован
					// Возвращаем статус в БД
					results <- OrderWithAccrual{
						Order:   order,
						Accrual: decimal.NewFromInt32(0),
						Status:  "REGISTERED",
					}
				case "PROCESSING":
					results <- OrderWithAccrual{
						Order:   order,
						Accrual: decimal.NewFromInt32(0),
						Status:  "PROCESSING",
					}
				case "INVALID":
					results <- OrderWithAccrual{
						Order:   order,
						Accrual: decimal.NewFromInt32(0),
						Status:  "INVALID",
					}
				}
			case http.StatusNoContent: //заказ не зарегистрирован в системе расчёта
				results <- OrderWithAccrual{
					Order:   order,
					Accrual: decimal.NewFromInt32(0),
					Status:  "NOT_REGISTRED",
				}
			case http.StatusTooManyRequests: // Превышено количество запросов к сервису
				// Запускаем механизм переотправки
				timeAdd, err := strconv.Atoi(resp.Header.Get("Retry-After"))
				if err != nil {
					return err
				}
				timeSec := time.Now().Add(time.Duration(timeAdd) * time.Second)

				// Сравниваем с текущей задержкой
				// Если пришло большее значение, то обновляем счетчик
				mu.Lock()
				if timeSec.After(*accrualMutex.LockedTime) {
					accrualMutex.LockedTime = &timeSec
				}
				mu.Unlock()

				go func(input InputAccrualType) {
					//засыпаем
					time.Sleep(time.Duration(timeAdd) * time.Second)
					select {
					case orders <- input:
					case <-ctx.Done():
						return
					}

				}(input)

			}
			resp.Body.Close()

		}
	}

}
