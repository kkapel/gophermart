package accrual

import (
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
)

type Accrual struct {
	URL string
	mu  sync.RWMutex
}

type AccrualResponse struct {
	Order   string `json:"order"`
	Status  string `json:"status"`
	Accrual int    `json:"accrual"` // Баллы accrual
}

type OrderWithAccrual struct {
	Order   string
	Accrual int
	Status  string
}

func NewAccrual(url string) *Accrual {
	return &Accrual{
		URL: url,
		mu:  sync.RWMutex{},
	}
}

// Функция отправки запроса в accrual
func SaveOrder(orders <-chan string, results chan<- OrderWithAccrual, urlAccrualBase string) error {

	order := <-orders
	accrualURL, err := url.JoinPath(urlAccrualBase, "/api/orders/", order)

	if err != nil {
		return err
	}

	resp, err := http.Get(accrualURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// читаем статус
	switch resp.Status {
	case "200":
		var accrualResponse AccrualResponse
		err = json.NewDecoder(resp.Body).Decode(&accrualResponse)

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
	case "204": //заказ не зарегистрирован в системе расчёта
		results <- OrderWithAccrual{
			Order:   order,
			Accrual: 0,
			Status:  "NOT_REGISTRED",
		}
	case "429": // Превышено количество запросов к сервису
	}

	return nil
}
