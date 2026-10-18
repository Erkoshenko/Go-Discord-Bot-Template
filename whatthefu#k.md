# Разбор кода бота: каждая строка

Oct 18, 2026 · @Erkoshh

## Как читать разбор

Каждый файл ниже показан целиком, а справа от строк кода стоит комментарий `// ...` с объяснением. Строки, где только закрывающая скобка `}`, не комментируются, кроме случаев, когда за ними есть что сказать. Пустые строки и строки `import` объяснены один раз в начале файла.

Порядок запуска бота (так же идёт и порядок разделов ниже):

1. `cmd/bot/main.go` читает настройки и запускает `App`.
2. `App` подключает базу и создаёт общие репозитории.
3. `App` создаёт модули (`general`, `moderation`) и собирает их команды в один реестр.
4. `App` создаёт сессию Discord, подписывает на события реестр и модули и подключается.
5. При запуске Discord присылает событие Ready, реестр регистрирует команды.
6. Пользователь вводит команду, реестр находит нужную и вызывает её `Execute`, команда вызывает сервис, сервис вызывает репозиторий и шлюз.

Идеи Go, которые встречаются в коде постоянно:

- **`package` и `import`.** Каждая папка это пакет. `bot/internal/config` это путь к пакету внутри модуля `bot` (имя берётся из `go.mod`). Имя с большой буквы (`Load`) видно из других пакетов, с маленькой (`validate`) нет.
- **`:=`** создаёт переменную и сразу присваивает значение: `x := 5`. Обычное `=` только меняет существующую.
- **Ошибки это значения.** Функция возвращает `(результат, error)`. Если `err != nil`, дальше идти нельзя. `fmt.Errorf("...: %w", err)` оборачивает ошибку с пояснением, а `errors.Is(err, X)` потом находит исходную.
- **`struct` и методы.** `func (s *Service) Warn(...)` это метод структуры `Service`; `s` внутри как `this`. `*Service` означает «указатель», то есть работа с одним и тем же объектом, а не с копией.
- **`interface`** это список методов. Любая структура, у которой такие методы есть, подходит под интерфейс автоматически, без слова `implements`. Поэтому в тестах вместо базы подставляется заглушка.
- **`context.Context`** переносит отмену и таймаут через вызовы: если время вышло, запрос к базе прерывается.
- **`defer`** откладывает вызов до выхода из функции: `defer db.Close()` закроет базу, как бы функция ни завершилась.
- **Горутины и каналы.** `go func(){...}()` запускает код параллельно, `chan` передаёт сигналы между ними, `select` ждёт первого из нескольких событий.
- **Встраивание.** Если в структуру вписать `module.Base` без имени поля, её методы становятся методами самой структуры. Так модули получают пустые реализации необязательных методов.

## cmd/bot/main.go

Точка входа. Здесь только жизненный цикл: прочитать настройки, запустить `App`, остановиться по сигналу или перезапуститься.

```go
package main // main — особое имя: пакет компилируется в исполняемую программу

import (
	"bot/internal/app"     // наш App, который собирает бота
	"bot/internal/config"  // загрузка настроек
	"context"              // контексты: отмена и таймауты
	"errors"               // errors.Is — проверка вида ошибки
	"fmt"                  // fmt.Errorf — создание ошибок с текстом
	"log/slog"             // структурные логи из стандартной библиотеки
	"os"                   // выход из программы, сигналы, файлы
	"os/signal"            // приём сигналов ОС (Ctrl+C и т.д.)
	"syscall"              // имена сигналов: SIGTERM, SIGHUP
	"time"                 // паузы и длительности

	"github.com/joho/godotenv" // чтение файла .env
)

func main() { // с этой функции начинается любая Go-программа
	log := slog.New(slog.NewTextHandler(os.Stdout, nil)) // логгер: пишет текстовые строки в стандартный вывод

	if err := run(log); err != nil { // запускаем всю работу; err виден только внутри этого if
		log.Error("бот остановлен с ошибкой", "err", err) // пишем причину в лог (пары ключ-значение)
		os.Exit(1) // выходим с кодом 1: для ОС и Docker это «завершилось с ошибкой»
	}
}

// run вынесен из main, чтобы defer-ы внутри успели отработать до os.Exit.
func run(log *slog.Logger) error { // возвращает ошибку или nil (nil = всё хорошо)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM) // ctx отменится при Ctrl+C или SIGTERM
	defer stop() // при выходе из run перестаём слушать эти сигналы

	hup := make(chan os.Signal, 1) // канал-«почтовый ящик» на 1 сигнал
	signal.Notify(hup, syscall.SIGHUP) // ОС будет класть сюда SIGHUP (команда перезапуска)
	defer signal.Stop(hup) // при выходе отписываемся от сигнала

	for { // бесконечный цикл: один оборот = один запуск бота
		// Настройки читаются на каждом обороте, поэтому перезапуск подхватывает правки.
		if err := loadEnv(); err != nil { // читаем .env
			return err // не смогли: выходим из run с этой ошибкой
		}
		settings, err := config.Load("configs/config.yaml") // читаем yaml и переменные окружения
		if err != nil {
			return fmt.Errorf("загрузка настроек: %w", err) // оборачиваем ошибку пояснением, %w сохраняет исходную
		}

		application := app.New(settings, log) // создаём App (пока ничего не запускает)
		runCtx, cancel := context.WithCancel(ctx) // дочерний контекст: отменим вручную после остановки App
		go func() { // горутина: параллельно ждёт SIGHUP
			select { // ждём первое из двух событий
			case <-hup: // пришёл SIGHUP
				log.Info("получен SIGHUP, перезапускаюсь")
				application.RequestRestart() // просим App завершиться с ErrRestart
			case <-runCtx.Done(): // App уже остановился, ждать больше нечего
			}
		}() // () в конце: функцию тут же вызываем

		log.Info("запуск бота")
		err = application.Run(runCtx) // ГЛАВНЫЙ вызов: работает, пока бот не остановят
		cancel() // освобождаем контекст и завершаем горутину выше

		switch { // switch без значения: берётся первое условие, равное true
		case errors.Is(err, app.ErrRestart): // App попросил перезапуск
			time.Sleep(time.Second) // секунда паузы, чтобы Discord закрыл старое соединение
			continue // начинаем цикл for заново
		case err != nil: // любая другая ошибка
			return err // выходим
		default: // err == nil: остановили Ctrl+C
			log.Info("бот остановлен")
			return nil // нормальный выход
		}
	}
}

// loadEnv читает .env. Файл необязателен: на сервере переменные задаёт окружение.
func loadEnv() error {
	if err := godotenv.Overload(); err != nil && !os.IsNotExist(err) { // Overload перезаписывает уже заданные переменные; "файла нет" ошибкой не считаем
		return fmt.Errorf("не удалось прочитать .env: %w", err)
	}
	return nil // успех или файла просто нет
}
```

## internal/app/app.go и modules.go

`App` собирает бота из частей и держит его запущенным. `modules.go` это список включённых модулей.

```go
package app // пакет app: сборка всего бота

import (
	"bot/internal/commands"   // реестр команд
	"bot/internal/config"     // тип Settings
	"bot/internal/database"   // подключение к Postgres
	"bot/internal/module"     // контракт модуля и Deps
	"bot/internal/repository" // общие репозитории
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo" // библиотека Discord
)

// ErrRestart возвращается из Run, когда кто-то запросил перезапуск.
var ErrRestart = errors.New("запрошен перезапуск") // готовое значение ошибки; по нему main узнаёт, что надо перезапуститься

type App struct { // App — структура с данными приложения
	settings config.Settings // настройки (токен, база, настройки модулей)
	log      *slog.Logger    // логгер
	restart  chan struct{}   // канал-сигнал «перезапустись»; struct{} — пустое значение, важен лишь факт отправки
}

func New(settings config.Settings, log *slog.Logger) *App { // конструктор: в Go так принято создавать структуры
	return &App{settings: settings, log: log, restart: make(chan struct{}, 1)} // &App{} — создаём и берём указатель; канал на 1 место
}

// RequestRestart просит Run завершиться с ErrRestart.
func (a *App) RequestRestart() { // метод App; a — это сам объект
	select { // неблокирующая отправка
	case a.restart <- struct{}{}: // если в канале есть место — кладём сигнал
	default: // иначе (сигнал уже лежит) ничего не делаем, чтобы вызов никогда не повис
	}
}

// Run собирает и запускает бота, блокируется до остановки или перезапуска.
func (a *App) Run(ctx context.Context) error {
	initCtx, cancel := context.WithTimeout(ctx, 15*time.Second) // контекст-«будильник»: на запуск даём 15 секунд
	defer cancel() // освободить ресурсы таймера при выходе

	// 1. База и общие репозитории.
	db, err := database.Connect(initCtx, a.settings.DB) // подключаемся к Postgres
	if err != nil {
		return err // не подключились: дальше смысла нет
	}
	defer db.Close() // закроем базу при выходе из Run (выполнится ПОСЛЕ закрытия Discord — defer идут в обратном порядке)

	repos := repository.NewRepositories(db) // создаём общие репозитории (users)
	if err := repos.CreateTables(initCtx); err != nil { // создаём их таблицы, если ещё нет
		return err
	}

	// 2. Модули.
	mods := buildModules(module.Deps{ // buildModules — из modules.go; Deps — то, что получает каждый модуль
		DB:             db,
		Repos:          repos,
		Settings:       a.settings,
		Log:            a.log,
		RequestRestart: a.RequestRestart, // передаём САМУ функцию (без скобок), модуль вызовет её сам, когда понадобится
	})

	registry := commands.NewRegistry(a.log, a.settings.TestGuildID) // пустой реестр команд
	intents := discordgo.IntentsGuilds // интенты — какие события Discord мы хотим получать; базовый: события серверов
	for _, m := range mods { // перебираем модули по очереди (_ — номер, он не нужен)
		if err := m.Init(initCtx); err != nil { // модуль создаёт свои таблицы
			return fmt.Errorf("модуль %s: %w", m.Name(), err) // в ошибке указываем, какой модуль сломался
		}
		if err := registry.Add(m.Commands()...); err != nil { // кладём команды модуля в реестр; ... раскрывает список в аргументы
			return fmt.Errorf("модуль %s: %w", m.Name(), err)
		}
		intents |= m.Intents() // |= добавляет к интентам те, что нужны модулю (побитовое ИЛИ)
		a.log.Info("модуль подключён", "module", m.Name())
	}

	// 3. Сессия Discord.
	sess, err := discordgo.New("Bot " + a.settings.DiscordToken) // сессия; префикс "Bot " обязателен для токена бота
	if err != nil {
		return fmt.Errorf("создание сессии Discord: %w", err)
	}
	sess.Identify.Intents = intents // говорим Discord, какие события хотим

	registry.Attach(sess) // реестр подписывается на Ready и на вызовы команд
	sess.AddHandler(func(_ *discordgo.Session, r *discordgo.Ready) { // подписка на событие Ready (бот вошёл); _ — сессия нам не нужна
		a.log.Info("бот запущен", "user", r.User.Username)
	})
	for _, m := range mods {
		m.Register(sess) // каждый модуль подписывается на свои события
	}

	if err := sess.Open(); err != nil { // подключаемся к Discord
		return fmt.Errorf("подключение к Discord: %w", err)
	}
	defer sess.Close() // закроем соединение при выходе (раньше, чем db.Close)

	// 4. Ждём остановки или запроса перезапуска.
	select { // ждём первое из двух событий
	case <-ctx.Done(): // нажали Ctrl+C
		return nil // штатная остановка
	case <-a.restart: // кто-то вызвал RequestRestart
		return ErrRestart // main увидит это и запустит всё заново
	}
}
```

```go
package app

import (
	"bot/internal/general"    // модуль с /ping
	"bot/internal/moderation" // модуль модерации
	"bot/internal/module"
)

// buildModules — единственное место, где включаются модули бота.
func buildModules(d module.Deps) []module.Module { // принимает общие зависимости, возвращает список модулей
	return []module.Module{ // []module.Module — срез (список) значений-модулей
		general.New(d),    // создаём модуль general
		moderation.New(d), // создаём модуль moderation; удалить строку = отключить модуль
	}
}
```

## internal/module/module.go

Контракт: что должен уметь любой модуль, чтобы `App` мог его подключить. Благодаря этому `App` ничего не знает о конкретных модулях.

```go
// Package module — контракт, который выполняет каждый модуль бота.
package module // комментарий перед package — описание пакета, его показывает go doc

import (
	"bot/internal/commands"   // тип Command
	"bot/internal/config"     // тип Settings
	"bot/internal/repository" // тип Repositories
	"context"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v5/pgxpool" // пул соединений с базой
)

// Deps — то, что App даёт каждому модулю при создании.
type Deps struct {
	DB             *pgxpool.Pool            // база (для собственных репозиториев модуля)
	Repos          *repository.Repositories // общие репозитории (users и др.)
	Settings       config.Settings          // все настройки; модуль берёт свой раздел
	Log            *slog.Logger             // логгер
	RequestRestart func()                   // функция: вызовите её, чтобы попросить перезапуск бота
}

// Module — самостоятельная часть бота (модерация, экономика, тикеты...).
type Module interface { // interface — набор методов, которые обязан иметь модуль
	Name() string // имя модуля для логов
	Init(ctx context.Context) error // вызывается один раз при старте: создать таблицы и т.п.; error если не вышло
	Commands() []commands.Command // слэш-команды модуля
	Intents() discordgo.Intent // дополнительные интенты (например, чтение сообщений)
	Register(s *discordgo.Session) // подписка на события Discord до подключения
}

// Base — пустая реализация необязательных методов.
type Base struct{} // пустая структура: полей нет, нужна только как носитель методов

func (Base) Init(context.Context) error   { return nil }       // (Base) — получатель без имени; параметр тоже без имени, он не нужен; всегда «успех»
func (Base) Commands() []commands.Command { return nil }       // по умолчанию команд нет
func (Base) Intents() discordgo.Intent    { return 0 }         // по умолчанию дополнительных интентов нет (0 — ни одного бита)
func (Base) Register(*discordgo.Session)  {}                   // по умолчанию на события не подписываемся
```

Важно: метод `Name()` в `Base` намеренно нет. Модуль обязан определить его сам, иначе он не подойдёт под `Module`, и компилятор не пропустит.

## internal/commands/registry.go

Реестр хранит команды всех модулей, регистрирует их в Discord и направляет вызовы нужной команде.

```go
// Package commands — каркас слэш-команд.
package commands

import (
	"bot/pkg/respond" // помощник ответа на команду
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

// Command — одна слэш-команда.
type Command interface {
	Definition() *discordgo.ApplicationCommand // описание для Discord: имя, описание, параметры, права
	Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error // код, который выполняется при вызове; s — сессия, i — данные вызова
}

// Registry хранит команды всех модулей...
type Registry struct {
	log     *slog.Logger
	guildID string             // пусто — глобальная регистрация, иначе только на этом сервере
	cmds    map[string]Command // словарь: имя команды -> команда
}

func NewRegistry(log *slog.Logger, guildID string) *Registry {
	return &Registry{log: log, guildID: guildID, cmds: make(map[string]Command)} // make создаёт пустой словарь (без make запись в него уронит программу)
}

// Add добавляет команды. Одинаковые имена у разных модулей — ошибка.
func (r *Registry) Add(cmds ...Command) error { // ...Command — любое число команд в аргументах (внутри — срез)
	for _, c := range cmds { // проходим по каждой
		name := c.Definition().Name // имя команды, например "warn"
		if _, exists := r.cmds[name]; exists { // есть ли уже такое имя: exists — true/false
			return fmt.Errorf("команда /%s зарегистрирована дважды", name) // два модуля не могут иметь одинаковую команду
		}
		r.cmds[name] = c // кладём команду в словарь по имени
	}
	return nil
}

// Attach подключает реестр к сессии.
func (r *Registry) Attach(s *discordgo.Session) {
	s.AddHandler(r.onReady)       // когда бот запустился — вызови onReady (метод без скобок — это ссылка на функцию)
	s.AddHandler(r.onInteraction) // когда пришло взаимодействие (вызов команды) — вызови onInteraction
}

func (r *Registry) onReady(s *discordgo.Session, _ *discordgo.Ready) { // discordgo сам по типу аргумента понимает, на какое событие это обработчик
	defs := make([]*discordgo.ApplicationCommand, 0, len(r.cmds)) // срез для описаний: длина 0, ёмкость по числу команд
	for _, c := range r.cmds { // проходим по словарю (_ — ключ-имя, нужны значения)
		defs = append(defs, c.Definition()) // append добавляет элемент в конец среза
	}
	if _, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, r.guildID, defs); err != nil { // одним запросом заменяем ВЕСЬ список команд бота; _ — возвращённый список нам не нужен
		r.log.Error("не удалось зарегистрировать команды", "err", err)
		return // в обработчике события нельзя вернуть ошибку, поэтому пишем в лог и выходим
	}
	r.log.Info("команды зарегистрированы", "count", len(defs))
}

func (r *Registry) onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand { // взаимодействие бывает разных типов: команда, кнопка, модалка...
		return // нам нужны только команды, остальное обрабатывают сами модули
	}

	name := i.ApplicationCommandData().Name // имя вызванной команды
	cmd, ok := r.cmds[name] // ищем в словаре; ok — нашли ли
	if !ok {
		return // неизвестная команда — игнорируем
	}

	// Паника в одной команде не должна ронять бота.
	defer func() { // отложенная анонимная функция, сработает при выходе из onInteraction
		if rec := recover(); rec != nil { // recover ловит панику; nil — паники не было
			r.log.Error("паника в команде", "command", name, "panic", rec)
			replyError(s, i) // отвечаем пользователю, чтобы не показывать «приложение не отвечает»
		}
	}()

	if err := cmd.Execute(s, i); err != nil { // запускаем команду
		r.log.Error("ошибка в команде", "command", name, "err", err)
		// Если команда уже ответила сама, Discord отклонит повторный ответ — это нормально.
		replyError(s, i)
	}
}

func replyError(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_ = respond.InteractionRespond(s, i, "❌ Что-то пошло не так. Попробуйте позже.", true) // _ = — осознанно игнорируем результат; true — ответ виден только вызвавшему
}
```
