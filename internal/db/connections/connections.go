package db

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type DB struct {
	db *sql.DB
}

// Инициализация БД
func InitDB(dbConnect string, migrationsPath string) (*DB, error) {
	if dbConnect == "" {
		return nil, nil
	}
	db, err := sql.Open("pgx", dbConnect)
	if err != nil {
		return nil, err
	}

	databaseInstance := &DB{db: db}

	if err := databaseInstance.CheckConnect(); err != nil {
		return nil, err
	}

	if err = migrateDB(dbConnect, migrationsPath); err != nil {
		return nil, err
	}

	return databaseInstance, nil

}

func (db *DB) CheckConnect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.db.PingContext(ctx); err != nil {
		return err
	}

	return nil
}

func (db *DB) Close() error {
	db.db.Close()
	return nil
}

func migrateDB(dbConnect string, migrationsPath string) error {
	m, err := migrate.New(migrationsPath, dbConnect)
	if err != nil {
		return err
	}
	if err = m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			return nil
		}
		if err.Error() == "Dirty database version 1. Fix and force version." ||
			strings.Contains(err.Error(), "dirty") {
			_ = m.Force(1) // Снимаем флаг dirty
			return m.Up()  // Пробуем накатить снова
		}
		return err
	}
	return nil
}

func (db *DB) GetSqlDb() *sql.DB {
	return db.db
}
