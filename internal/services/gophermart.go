package services

import (
	db "gophermart/internal/db/connections"
)

type GophermartService struct {
	db *db.DB
}

func CreateGophermartService(db *db.DB) *GophermartService {
	return &GophermartService{
		db: db,
	}

}

func RegisterUser(login string, password string) {

	// формирование хеша для пароля
	//hashPassword := HashPassword(password)
	// Вызов БД
}
