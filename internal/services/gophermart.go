package services

import (
	"context"
	"database/sql"
	db "gophermart/internal/db"
)

type GophermartService struct {
	queries *db.Queries
}

func CreateGophermartService(conn *sql.DB) *GophermartService {
	return &GophermartService{
		queries: db.New(conn),
	}

}

func (s *GophermartService) RegisterUser(ctx context.Context, login string, password string) error {

	// формирование хеша для пароля
	hashPassword, err := HashPassword(password)
	if err != nil {
		return err
	}

	// Вызов БД
	err = s.queries.SaveUser(ctx, db.SaveUserParams{
		Login:    login,
		PassHash: hashPassword,
	})

	return err

}
