package moderation

import (
	"bot/internal/config"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

const (
	MaxTimeout     = 28 * 24 * time.Hour // лимит Discord
	MaxReasonRunes = 400
	DefaultReason  = "не указана"
)

var (
	ErrInvalidDuration = errors.New("длительность мута должна быть от 1 минуты до 28 дней")
	// ErrActionFailed — Discord отказал (нет прав у бота, роль выше и т.п.). Запись не создаётся.
	ErrActionFailed = errors.New("действие в Discord не выполнено")
	// ErrNotRecorded — действие в Discord выполнено, но записать его в базу не удалось.
	ErrNotRecorded = errors.New("действие выполнено, но не записано в историю")
)

// Request — кого и за что наказываем. К моменту вызова сервиса права модератора
// и иерархия ролей уже проверены в слое команд.
type Request struct {
	GuildID     string
	TargetID    string
	ModeratorID string
	Reason      string
}

// WarnResult — итог выдачи предупреждения.
type WarnResult struct {
	Case        Case
	ActiveWarns int   // сколько действующих предупреждений теперь у пользователя
	AutoTimeout *Case // не nil, если предупреждение привело к автомуту
}

// Service — бизнес-логика модерации: правила, порядок действий, эскалация.
// Про discordgo и SQL не знает: работает через Gateway и Repository.
type Service struct {
	repo Repository
	gw   Gateway
	cfg  config.Moderation
	log  *slog.Logger
	now  func() time.Time
}

func NewService(repo Repository, gw Gateway, cfg config.Moderation, log *slog.Logger) *Service {
	return &Service{repo: repo, gw: gw, cfg: cfg, log: log, now: time.Now}
}

// Warn выдаёт предупреждение. Если действующих предупреждений набралось
// warn_limit или больше, выдаёт автоматический мут.
func (s *Service) Warn(ctx context.Context, req Request) (WarnResult, error) {
	c := s.newCase(req, ActionWarn)
	id, err := s.repo.AddCase(ctx, c)
	if err != nil {
		return WarnResult{}, err
	}
	c.ID = id
	res := WarnResult{Case: c}

	if n, err := s.repo.CountActiveWarns(ctx, req.GuildID, req.TargetID); err != nil {
		s.log.Warn("не удалось посчитать предупреждения", "err", err)
	} else {
		res.ActiveWarns = n
	}

	s.gw.DM(req.TargetID, dmWarn(c.Reason))
	s.gw.Log(logCase(c, "Предупреждение", ""))

	if s.cfg.WarnLimit > 0 && s.cfg.WarnTimeoutMinutes > 0 && res.ActiveWarns >= s.cfg.WarnLimit {
		auto := Request{
			GuildID:     req.GuildID,
			TargetID:    req.TargetID,
			ModeratorID: s.gw.BotID(),
			Reason:      fmt.Sprintf("автоматически: действующих предупреждений — %d", res.ActiveWarns),
		}
		tc, err := s.Timeout(ctx, auto, time.Duration(s.cfg.WarnTimeoutMinutes)*time.Minute)
		if err != nil {
			s.log.Warn("не удалось выдать автомут", "err", err)
		} else {
			res.AutoTimeout = &tc
		}
	}
	return res, nil
}

// Unwarn снимает предупреждение. false — предупреждение не найдено или уже снято.
func (s *Service) Unwarn(ctx context.Context, guildID string, caseID int64, moderatorID string) (bool, error) {
	ok, err := s.repo.DeactivateWarn(ctx, guildID, caseID)
	if err != nil || !ok {
		return ok, err
	}
	s.gw.Log(fmt.Sprintf("📋 Предупреждение #%d снято модератором <@%s>", caseID, moderatorID))
	return true, nil
}

// History возвращает последние записи пользователя.
func (s *Service) History(ctx context.Context, guildID, userID string, limit int) ([]Case, error) {
	return s.repo.GetCases(ctx, guildID, userID, limit)
}

// Timeout выдаёт мут на срок d.
func (s *Service) Timeout(ctx context.Context, req Request, d time.Duration) (Case, error) {
	if d < time.Minute || d > MaxTimeout {
		return Case{}, ErrInvalidDuration
	}
	until := s.now().Add(d)

	if err := s.gw.Timeout(req.GuildID, req.TargetID, until); err != nil {
		return Case{}, fmt.Errorf("%w: %w", ErrActionFailed, err)
	}

	c := s.newCase(req, ActionTimeout)
	c.ExpiresAt = &until
	if err := s.record(ctx, &c); err != nil {
		return c, err
	}

	s.gw.DM(req.TargetID, dmTimeout(until.Unix(), c.Reason))
	s.gw.Log(logCase(c, "Мут", fmt.Sprintf("До: <t:%d:f>\n", until.Unix())))
	return c, nil
}

// Kick выгоняет пользователя с сервера.
func (s *Service) Kick(ctx context.Context, req Request) (Case, error) {
	c := s.newCase(req, ActionKick)

	// Личное сообщение нужно отправить до кика: потом общих серверов может не остаться.
	s.gw.DM(req.TargetID, dmKick(c.Reason))
	if err := s.gw.Kick(req.GuildID, req.TargetID, c.Reason); err != nil {
		return Case{}, fmt.Errorf("%w: %w", ErrActionFailed, err)
	}

	if err := s.record(ctx, &c); err != nil {
		return c, err
	}
	s.gw.Log(logCase(c, "Кик", ""))
	return c, nil
}

// Ban банит пользователя и удаляет его сообщения за последние deleteDays дней (0–7).
func (s *Service) Ban(ctx context.Context, req Request, deleteDays int) (Case, error) {
	c := s.newCase(req, ActionBan)
	deleteDays = max(0, min(deleteDays, 7))

	s.gw.DM(req.TargetID, dmBan(c.Reason))
	if err := s.gw.Ban(req.GuildID, req.TargetID, c.Reason, deleteDays); err != nil {
		return Case{}, fmt.Errorf("%w: %w", ErrActionFailed, err)
	}

	if err := s.record(ctx, &c); err != nil {
		return c, err
	}
	s.gw.Log(logCase(c, "Бан", ""))
	return c, nil
}

// record сохраняет дело и проставляет ему номер.
func (s *Service) record(ctx context.Context, c *Case) error {
	id, err := s.repo.AddCase(ctx, *c)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrNotRecorded, err)
	}
	c.ID = id
	return nil
}

// newCase заполняет дело; пустую причину заменяет, слишком длинную обрезает.
func (s *Service) newCase(req Request, action Action) Case {
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = DefaultReason
	}
	if r := []rune(reason); len(r) > MaxReasonRunes {
		reason = string(r[:MaxReasonRunes-1]) + "…"
	}
	return Case{
		GuildID:     req.GuildID,
		UserID:      req.TargetID,
		ModeratorID: req.ModeratorID,
		Action:      action,
		Reason:      reason,
		Active:      true,
	}
}
