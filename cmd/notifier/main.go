package main

import (
	"database/sql"
	"fmt"
	"github.com/kotche/bot/internal/app/notifier"
	"github.com/kotche/bot/internal/config"
	"github.com/kotche/bot/internal/metrics"
	notes_repo "github.com/kotche/bot/internal/repository/notes"
	"github.com/kotche/bot/internal/service/kafka"
	notes_serv "github.com/kotche/bot/internal/service/notes"
	"log"
	"time"

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
	metrics.StartMetricsServer(":8081")

	bot, err := telebot.NewBot(telebot.Settings{
		Token:  cfg.TelegramConfig.TokenNotifyBot,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Fatal(err)
	}

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.PostgresConfig.Host,
		cfg.PostgresConfig.Port,
		cfg.PostgresConfig.User,
		cfg.PostgresConfig.Password,
		cfg.PostgresConfig.DBName,
		cfg.PostgresConfig.SSLMode,
	)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalln(err)
	}

	kafkaServ, err := kafka.New(
		cfg.KafkaConfig.Brokers,
		cfg.KafkaConfig.Topic,
		cfg.KafkaConfig.GroupID,
		1,
		1,
	)
	if err != nil {
		log.Fatalf("failed to initialize kafka: %v", err)
	}
	defer kafkaServ.Close()

	notesServ := notes_serv.NewDefaultService(notes_repo.NewDefaultRepository(db))
	notifierImpl := notifier.New(bot, notesServ, kafkaServ)
	notifierImpl.Start()
}
