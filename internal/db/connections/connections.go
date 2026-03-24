package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5"
)

type DB struct {
	db *sql.DB
}

// Инициализация БД
func InitDB(dbConnect string) (*DB, error) {
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

	if err = migrateDB(dbConnect); err != nil {
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

func migrateDB(dbConnect string) error {
	m, err := migrate.New("file://migrations", dbConnect)
	if err != nil {
		return err
	}
	if err = m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

func (db *DB) GetSqlDb() *sql.DB {
	return db.db
}
