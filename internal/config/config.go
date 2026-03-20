package config

import "github.com/spf13/viper"

type Config struct {
	RunAddress           string
	DataBaseURI          string
	AccrualSystemAddress string
}

func CreateConfig() *Config {
	//Используем Viper для получения переменных окружения
	viper.AllowEmptyEnv(true)
	viper.AutomaticEnv()
	runAddress := viper.GetString("RUN_ADDRESS")
	dataBaseURI := viper.GetString("DATABASE_URI")
	accrualSystemAddress := viper.GetString("ACCRUAL_SYSTEM_ADDRESS")

	return &Config{
		RunAddress:           "",
		DataBaseURI:          "",
		AccrualSystemAddress: "",
	}
}
