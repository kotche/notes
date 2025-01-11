package main

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/kotche/bot/infrastructure/metrics"
	"github.com/kotche/bot/infrastructure/tracing"
	"github.com/kotche/bot/internal/app/writer"
	"github.com/kotche/bot/internal/config"
	notes_repo "github.com/kotche/bot/internal/repository/notes"
	notes_serv "github.com/kotche/bot/internal/service/notes"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

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
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	metrics.Init()
	metrics.StartMetricsServer(":8080")

	bot, err := telebot.NewBot(telebot.Settings{
		Token:  cfg.TelegramConfig.TokenWriteBot,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Fatal(err)
	}

	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.PostgresConfig.User,
		cfg.PostgresConfig.Password,
		cfg.PostgresConfig.Host,
		cfg.PostgresConfig.Port,
		cfg.PostgresConfig.DBName,
		cfg.PostgresConfig.SSLMode,
	)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalln(err)
	}

	if err = runMigrations(connStr); err != nil {
		log.Fatalln("migration error:", err)
	}

	_, cleanup, err := tracing.InitTracing(cfg.TracingConfig.Endpoint)
	if err != nil {
		log.Fatal(err)
	}
	defer cleanup()

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
