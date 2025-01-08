package main

import (
	"database/sql"
	"github.com/kotche/bot/internal/app/notifier"
	notes_repo "github.com/kotche/bot/internal/repository/notes"
	"github.com/kotche/bot/internal/service/kafka"
	notes_serv "github.com/kotche/bot/internal/service/notes"
	"log"
	"os"
	"time"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/joho/godotenv"
	"gopkg.in/telebot.v3"
)

func init() {
	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Fatalf("failed to load location: %v", err)
	}
	time.Local = location
	log.Println("default time zone set to Europe/Moscow")
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalln("loading .env error: ", err)
	}

	token := os.Getenv("TOKEN_NOTIFY_BOT")
	if token == "" {
		log.Fatalln("TOKEN_NOTIFY_BOT is not read")
	}

	bot, err := telebot.NewBot(telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Fatal(err)
	}

	//TODO(cheki) вынести креды в env
	//connStr := "host=localhost port=5432 user=youruser password=yourpassword dbname=yourdb sslmode=disable"
	connStr := "postgres://youruser:yourpassword@localhost:5432/yourdb?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalln(err)
	}

	notesServ := notes_serv.NewDefaultService(notes_repo.NewDefaultRepository(db))

	kafkaServ, err := kafka.New([]string{"localhost:9092"}, "notifications",
		"notification-consumers", 1, 1)
	if err != nil {
		log.Fatalf("failed to initialize kafka: %v", err)
	}
	defer kafkaServ.Close()

	notifierImpl := notifier.New(bot, notesServ, kafkaServ)
	notifierImpl.Start()
}
