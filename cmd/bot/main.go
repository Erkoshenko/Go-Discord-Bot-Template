package main

import (
	"bot/internal/config"
	"bot/internal/database"
	"bot/internal/handlers"
	"bot/internal/repository"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

func main() {
	settings, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("ошибка загрузки настроек: %v", err)
	}

	db, err := database.InitDB(settings.DB)
	if err != nil {
		log.Fatalln(err)
	}

	repos := repository.NewRepositories(db)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	repos.Users.CreateTableIfNotExists(ctx)

	h := handlers.NewHandler(repos, settings)

	sess, err := discordgo.New("Bot " + settings.DiscordToken)
	if err != nil {
		fmt.Println("ошибка создании сессий: " + err.Error())
		return
	}

	h.RegisterAll(sess)

	err = sess.Open()
	if err != nil {
		fmt.Println("ошибка открытии сессий: " + err.Error())
		return
	}

	defer sess.Close()

	fmt.Println("Нажмите ctrl+c чтобы убить")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc
}
