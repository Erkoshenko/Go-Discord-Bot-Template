package database

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	dbPool *pgxpool.Pool
	once   sync.Once
)

type PGConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	DBName   string
}

func InitDB(cfg PGConfig) (*pgxpool.Pool, error) {
	var err error
	once.Do(func() {
		dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require",
			cfg.User,
			url.QueryEscape(cfg.Password),
			cfg.Host,
			cfg.Port,
			cfg.DBName,
		)

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		var poolConfig *pgxpool.Config
		poolConfig, err = pgxpool.ParseConfig(dsn)
		if err != nil {
			err = fmt.Errorf("ошибка парсинга конфига PG: %w", err)
			return
		}

		poolConfig.MaxConns = 3
		poolConfig.MinConns = 1
		poolConfig.MaxConnLifetime = 5 * time.Minute

		dbPool, err = pgxpool.NewWithConfig(ctx, poolConfig)
		if err != nil {
			err = fmt.Errorf("ошибка при созданий пула PG: %w", err)
			return
		}

		if err = dbPool.Ping(ctx); err != nil {
			dbPool.Close()
			err = fmt.Errorf("ошибка пинга PG: %w", err)
			return
		}

		log.Println("Успешное подключение к Postgres базе данных")
	})

	return dbPool, err
}

func GetDB() *pgxpool.Pool {
	if dbPool == nil {
		log.Panic("База данных не иницировано")
	}
	return dbPool
}

func CloseDB() {
	if dbPool != nil {
		dbPool.Close()
		log.Println("Соединение с MySQL закрыто.")
	}
}
