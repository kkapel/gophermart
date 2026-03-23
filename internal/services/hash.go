package services

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	return string(bytes), err
}

// Функция проверки пароля с использованием либы bcrypt
func CheckPassword(requestPassword string, passwordFromDb string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(passwordFromDb), []byte(passwordFromDb))

	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}
