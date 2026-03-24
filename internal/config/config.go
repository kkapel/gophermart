package config

import (
	"errors"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	RunAddress           string
	DataBaseURI          string
	AccrualSystemAddress string
}

var ErrFlagAndEnvVarNotFound = errors.New("Flag and environment variables not found")

func CreateConfig() (*Config, error) {
	//Используем Viper для получения флагов и переменных окружения
	pflag.String("a", "localhost:8080", "Run address flag")
	pflag.String("d", "", "Database uri flag")
	pflag.String("r", "localhost:8080", "Accrual system address")

	if !pflag.CommandLine.Parsed() {
		pflag.Parse()
	}

	viper.BindPFlags(pflag.CommandLine)
	runAddressFlag := viper.GetString("a")
	databaseURIFlag := viper.GetString("d")
	accrualSystemAddressFlag := viper.GetString("r")

	// Переменные окружения
	viper.AllowEmptyEnv(true)
	viper.AutomaticEnv()
	runAddressEnv := viper.GetString("RUN_ADDRESS")
	dataBaseURIEnv := viper.GetString("DATABASE_URI")
	accrualSystemAddressEnv := viper.GetString("ACCRUAL_SYSTEM_ADDRESS")

	// Приоритет:
	// 1. Флаги
	// 2. Переменные окружения
	var runAddress, dataBaseURI, accrualSystemAddress string
	if runAddressFlag != "" {
		runAddress = runAddressFlag
	} else if runAddressEnv != "" {
		runAddress = runAddressEnv
	}

	if databaseURIFlag != "" {
		dataBaseURI = databaseURIFlag
	} else if dataBaseURIEnv != "" {
		dataBaseURI = dataBaseURIEnv
	} else {
		return nil, ErrFlagAndEnvVarNotFound
	}

	if accrualSystemAddressFlag != "" {
		accrualSystemAddress = accrualSystemAddressFlag
	} else if accrualSystemAddressEnv != "" {
		accrualSystemAddress = accrualSystemAddressEnv
	}

	return &Config{
		RunAddress:           runAddress,
		DataBaseURI:          dataBaseURI,
		AccrualSystemAddress: accrualSystemAddress,
	}, nil
}
