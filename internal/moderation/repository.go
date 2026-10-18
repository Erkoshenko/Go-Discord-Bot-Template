package moderation

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository — хранилище модуля. Только SQL, ничего про Discord.
// Таблица принадлежит модулю: другие модули к ней не обращаются.
type Repository interface {
	CreateTableIfNotExists(ctx context.Context) error
	// AddCase сохраняет запись и возвращает её номер.
	AddCase(ctx context.Context, c Case) (int64, error)
	// GetCases возвращает последние записи пользователя на сервере (новые первыми).
	GetCases(ctx context.Context, guildID, userID string, limit int) ([]Case, error)
	// CountActiveWarns считает действующие предупреждения.
	CountActiveWarns(ctx context.Context, guildID, userID string) (int, error)
	// DeactivateWarn снимает предупреждение. false — если такого нет или оно уже снято.
	DeactivateWarn(ctx context.Context, guildID string, caseID int64) (bool, error)
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) CreateTableIfNotExists(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS moderation_cases(
			id           BIGSERIAL PRIMARY KEY,
			guild_id     TEXT NOT NULL,
			user_id      TEXT NOT NULL,
			moderator_id TEXT NOT NULL,
			action       TEXT NOT NULL,
			reason       TEXT NOT NULL DEFAULT '',
			active       BOOLEAN NOT NULL DEFAULT TRUE,
			created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
			expires_at   TIMESTAMPTZ
		)`,
		`CREATE INDEX IF NOT EXISTS idx_moderation_cases_user
			ON moderation_cases (guild_id, user_id, created_at DESC)`,
	}
	for _, q := range statements {
		if _, err := r.db.Exec(ctx, q); err != nil {
			return fmt.Errorf("не удалось создать таблицу модерации: %w", err)
		}
	}
	return nil
}

func (r *postgresRepository) AddCase(ctx context.Context, c Case) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `
		INSERT INTO moderation_cases (guild_id, user_id, moderator_id, action, reason, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		c.GuildID, c.UserID, c.ModeratorID, string(c.Action), c.Reason, c.ExpiresAt,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("не удалось сохранить запись модерации: %w", err)
	}
	return id, nil
}

func (r *postgresRepository) GetCases(ctx context.Context, guildID, userID string, limit int) ([]Case, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, guild_id, user_id, moderator_id, action, reason, active, created_at, expires_at
		FROM moderation_cases
		WHERE guild_id = $1 AND user_id = $2
		ORDER BY created_at DESC, id DESC
		LIMIT $3`,
		guildID, userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить историю модерации: %w", err)
	}
	defer rows.Close()

	var cases []Case
	for rows.Next() {
		var c Case
		var action string
		if err := rows.Scan(&c.ID, &c.GuildID, &c.UserID, &c.ModeratorID, &action,
			&c.Reason, &c.Active, &c.CreatedAt, &c.ExpiresAt); err != nil {
			return nil, fmt.Errorf("не удалось прочитать запись модерации: %w", err)
		}
		c.Action = Action(action)
		cases = append(cases, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка чтения истории модерации: %w", err)
	}
	return cases, nil
}

func (r *postgresRepository) CountActiveWarns(ctx context.Context, guildID, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM moderation_cases
		WHERE guild_id = $1 AND user_id = $2 AND action = 'warn' AND active`,
		guildID, userID,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("не удалось посчитать предупреждения: %w", err)
	}
	return n, nil
}

func (r *postgresRepository) DeactivateWarn(ctx context.Context, guildID string, caseID int64) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE moderation_cases SET active = FALSE
		WHERE id = $1 AND guild_id = $2 AND action = 'warn' AND active`,
		caseID, guildID,
	)
	if err != nil {
		return false, fmt.Errorf("не удалось снять предупреждение: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}
