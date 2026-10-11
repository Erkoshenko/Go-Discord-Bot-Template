package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Settings struct {
	DiscordToken string   `yaml:"-"`
	DB           PGConfig `yaml:"-"`
	TestGuildID  string   `yaml:"test_guild_id"`
}

type PGConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	DBName   string
}

func Load(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Settings{}, fmt.Errorf("не удалось прочитать %s: %w", path, err)
	}

	var s Settings
	if err := yaml.Unmarshal(data, &s); err != nil {
		return Settings{}, fmt.Errorf("не удалось обработать данные %s: %w", path, err)
	}

	s.DiscordToken = os.Getenv("DISCORD_TOKEN")
	s.DB = PGConfig{
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		DBName:   os.Getenv("DB_NAME"),
	}

	if s.DiscordToken == "" {
		log.Fatal("не задана переменная окружения TOKEN")
	}

	return s, nil
}
