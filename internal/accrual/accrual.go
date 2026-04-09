package accrual

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
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
	Order   string `json:"order"`
	Status  string `json:"status"`
	Accrual int    `json:"accrual"` // Баллы accrual
}

type OrderWithAccrual struct {
	Order   string
	Accrual int32
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
func SaveOrder(orders chan InputAccrualType, results chan<- OrderWithAccrual, errors chan<- error, urlAccrualBase string) {

	for input := range orders {
		func() {
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
				errors <- err
				return
			}

			resp, err := http.Get(accrualURL)
			if err != nil {
				errors <- err
				return
			}
			defer resp.Body.Close()

			// читаем статус
			switch resp.StatusCode {
			case http.StatusOK:
				var accrualResponse AccrualResponse
				err = json.NewDecoder(resp.Body).Decode(&accrualResponse)
				if err != nil {
					errors <- err
					return
				}
				switch accrualResponse.Status {
				case "PROCESSED":
					//если пришел успешный ответ и баллы
					// Пишем в канал результатов
					results <- OrderWithAccrual{
						Order:   order,
						Accrual: int32(accrualResponse.Accrual),
						Status:  "PROCESSED",
					}
				case "REGISTERED":
					// Заказ зарегистрирован
					// Возвращаем статус в БД
					results <- OrderWithAccrual{
						Order:   order,
						Accrual: 0,
						Status:  "REGISTERED",
					}
				case "PROCESSING":
					results <- OrderWithAccrual{
						Order:   order,
						Accrual: 0,
						Status:  "PROCESSING",
					}
				case "INVALID":
					results <- OrderWithAccrual{
						Order:   order,
						Accrual: 0,
						Status:  "INVALID",
					}
				}
			case http.StatusNoContent: //заказ не зарегистрирован в системе расчёта
				results <- OrderWithAccrual{
					Order:   order,
					Accrual: 0,
					Status:  "NOT_REGISTRED",
				}
			case http.StatusTooManyRequests: // Превышено количество запросов к сервису
				// Запускаем механизм переотправки
				timeAdd, err := strconv.Atoi(resp.Header.Get("Retry-After"))
				if err != nil {
					errors <- err
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
					orders <- input
				}(input)

			}

		}()

	}
}
