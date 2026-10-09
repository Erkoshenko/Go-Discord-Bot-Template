package config

import (
	"bot/internal/database"
	"os"

	"github.com/joho/godotenv"
)

type BotConfig struct {
	Token    string
	DBConfig database.PGConfig
}

func LoadConfig() BotConfig {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	return BotConfig{
		Token: os.Getenv("TOKEN"),
		DBConfig: database.PGConfig{
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			DBName:   os.Getenv("DB_NAME"),
		},
	}
}
