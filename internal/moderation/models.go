package moderation

import "time"

// Action — тип модераторского действия.
type Action string

const (
	ActionWarn    Action = "warn"
	ActionTimeout Action = "timeout"
	ActionKick    Action = "kick"
	ActionBan     Action = "ban"
)

// Case — одна запись в истории модерации («дело»).
type Case struct {
	ID          int64
	GuildID     string
	UserID      string
	ModeratorID string
	Action      Action
	Reason      string
	Active      bool // для warn: false, если предупреждение снято
	CreatedAt   time.Time
	ExpiresAt   *time.Time // для timeout: когда мут закончится
}
