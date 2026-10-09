package repository

import "github.com/jackc/pgx/v5/pgxpool"

type Repositories struct {
	Users *UserRepository
}

func NewRepositories(db *pgxpool.Pool) *Repositories {
	return &Repositories{
		Users: NewUserRepository(db),
	}
}
