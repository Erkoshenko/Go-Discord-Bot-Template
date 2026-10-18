package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Settings собирается из двух источников:
//   - configs/config.yaml — несекретные настройки (ID сервера, настройки модулей);
//   - переменные окружения (.env) — токен и параметры базы.
type Settings struct {
	DiscordToken string   `yaml:"-"`
	DB           PGConfig `yaml:"-"`
	TestGuildID  string   `yaml:"test_guild_id"`

	// Настройки модулей. Новый модуль — новое поле и структура ниже.
	Moderation Moderation `yaml:"moderation"`
}

// Moderation — настройки модуля internal/moderation.
type Moderation struct {
	LogChannelID       string `yaml:"log_channel"`          // канал журнала; пусто — журнала нет
	WarnLimit          int    `yaml:"warn_limit"`           // сколько предупреждений ведёт к автомуту; 0 — выключено
	WarnTimeoutMinutes int    `yaml:"warn_timeout_minutes"` // длительность автомута
}

type PGConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	DBName   string
	SSLMode  string // disable | prefer | require | verify-full
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
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  getEnv("DB_SSLMODE", "prefer"),
	}

	if err := s.validate(); err != nil {
		return Settings{}, err
	}
	return s, nil
}

func (s Settings) validate() error {
	var missing []string
	if s.DiscordToken == "" {
		missing = append(missing, "DISCORD_TOKEN")
	}
	if s.DB.User == "" {
		missing = append(missing, "DB_USER")
	}
	if s.DB.DBName == "" {
		missing = append(missing, "DB_NAME")
	}
	if len(missing) > 0 {
		return errors.New("не заданы переменные окружения: " + strings.Join(missing, ", "))
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
