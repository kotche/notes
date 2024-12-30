package main

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/kotche/bot/internal/app/writer"
	notes_repo "github.com/kotche/bot/internal/repository/notes"
	notes_serv "github.com/kotche/bot/internal/service/notes"
	"log"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
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

	token := os.Getenv("TOKEN_WRITE_BOT")
	if token == "" {
		log.Fatalln("TOKEN_WRITE_BOT is not read")
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

	if err = runMigrations(connStr); err != nil {
		log.Fatalln("migration error:", err)
	}

	notesServ := notes_serv.NewDefaultService(notes_repo.NewDefaultRepository(db))

	writerImpl := writer.New(bot, notesServ)
	writerImpl.Start()
}

func runMigrations(dbURL string) error {
	m, err := migrate.New(
		"file://migrations",
		dbURL,
	)
	if err != nil {
		return fmt.Errorf("failed to init migrations: %w", err)
	}

	if err = m.Up(); !errors.Is(err, migrate.ErrNoChange) && err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}
