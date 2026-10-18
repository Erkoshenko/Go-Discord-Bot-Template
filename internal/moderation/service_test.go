package moderation

import (
	"bot/internal/config"
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
)

// Пример теста бизнес-логики: вместо базы и Discord подставлены заглушки.

type fakeRepo struct {
	cases []Case
	warns int
}

func (r *fakeRepo) CreateTableIfNotExists(context.Context) error { return nil }
func (r *fakeRepo) AddCase(_ context.Context, c Case) (int64, error) {
	r.cases = append(r.cases, c)
	if c.Action == ActionWarn {
		r.warns++
	}
	return int64(len(r.cases)), nil
}
func (r *fakeRepo) GetCases(context.Context, string, string, int) ([]Case, error) {
	return r.cases, nil
}
func (r *fakeRepo) CountActiveWarns(context.Context, string, string) (int, error) {
	return r.warns, nil
}
func (r *fakeRepo) DeactivateWarn(context.Context, string, int64) (bool, error) { return true, nil }

type fakeGateway struct {
	timeouts int
	kicks    int
	failNext error
}

func (g *fakeGateway) BotID() string { return "bot" }
func (g *fakeGateway) Timeout(string, string, time.Time) error {
	g.timeouts++
	return g.failNext
}
func (g *fakeGateway) Kick(string, string, string) error     { g.kicks++; return g.failNext }
func (g *fakeGateway) Ban(string, string, string, int) error { return g.failNext }
func (g *fakeGateway) DM(string, string)                     {}
func (g *fakeGateway) Log(string)                            {}

func newTestService(cfg config.Moderation) (*Service, *fakeRepo, *fakeGateway) {
	repo, gw := &fakeRepo{}, &fakeGateway{}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewService(repo, gw, cfg, log), repo, gw
}

var req = Request{GuildID: "g", TargetID: "u", ModeratorID: "m", Reason: "спам"}

func TestWarnAutoTimeoutAtLimit(t *testing.T) {
	svc, _, gw := newTestService(config.Moderation{WarnLimit: 2, WarnTimeoutMinutes: 60})

	res, err := svc.Warn(context.Background(), req)
	if err != nil || res.AutoTimeout != nil {
		t.Fatalf("первое предупреждение: err=%v, автомут=%v (ожидали nil)", err, res.AutoTimeout)
	}

	res, err = svc.Warn(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if res.AutoTimeout == nil || gw.timeouts != 1 {
		t.Fatalf("второе предупреждение должно дать автомут: %+v, вызовов мута: %d", res, gw.timeouts)
	}
}

func TestWarnLimitDisabled(t *testing.T) {
	svc, _, gw := newTestService(config.Moderation{})
	for range 5 {
		if _, err := svc.Warn(context.Background(), req); err != nil {
			t.Fatal(err)
		}
	}
	if gw.timeouts != 0 {
		t.Fatalf("автомут выключен, а мутов выдано: %d", gw.timeouts)
	}
}

func TestTimeoutInvalidDuration(t *testing.T) {
	svc, repo, _ := newTestService(config.Moderation{})
	for _, d := range []time.Duration{0, 30 * time.Second, MaxTimeout + time.Minute} {
		if _, err := svc.Timeout(context.Background(), req, d); !errors.Is(err, ErrInvalidDuration) {
			t.Errorf("длительность %v: ожидали ErrInvalidDuration, получили %v", d, err)
		}
	}
	if len(repo.cases) != 0 {
		t.Error("при неверной длительности запись создаваться не должна")
	}
}

func TestKickDiscordFailureNotRecorded(t *testing.T) {
	svc, repo, gw := newTestService(config.Moderation{})
	gw.failNext = errors.New("missing permissions")

	_, err := svc.Kick(context.Background(), req)
	if !errors.Is(err, ErrActionFailed) {
		t.Fatalf("ожидали ErrActionFailed, получили %v", err)
	}
	if len(repo.cases) != 0 {
		t.Error("если Discord отказал, запись создаваться не должна")
	}
}

func TestReasonDefaultAndTruncate(t *testing.T) {
	svc, _, _ := newTestService(config.Moderation{})

	c := svc.newCase(Request{Reason: "   "}, ActionWarn)
	if c.Reason != DefaultReason {
		t.Errorf("пустая причина: получили %q", c.Reason)
	}

	long := make([]rune, MaxReasonRunes+50)
	for i := range long {
		long[i] = 'я'
	}
	c = svc.newCase(Request{Reason: string(long)}, ActionWarn)
	if n := len([]rune(c.Reason)); n != MaxReasonRunes {
		t.Errorf("длина причины после обрезки: %d, ожидали %d", n, MaxReasonRunes)
	}
}
