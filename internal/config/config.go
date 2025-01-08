package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramConfig TelegramConfig
	PostgresConfig PostgresConfig
	KafkaConfig    KafkaConfig
}

type TelegramConfig struct {
	TokenWriteBot  string
	TokenNotifyBot string
}

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type KafkaConfig struct {
	Brokers []string
	Topic   string
	GroupID string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not found, using environment variables")
	}

	config := &Config{
		TelegramConfig: TelegramConfig{
			TokenWriteBot:  getEnv("TOKEN_WRITE_BOT", ""),
			TokenNotifyBot: getEnv("TOKEN_NOTIFY_BOT", ""),
		},
		PostgresConfig: PostgresConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnv("POSTGRES_PORT", "5432"),
			User:     getEnv("POSTGRES_USER", "user"),
			Password: getEnv("POSTGRES_PASSWORD", "password"),
			DBName:   getEnv("POSTGRES_DB", "dbname"),
			SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
		},
		KafkaConfig: KafkaConfig{
			Brokers: []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
			Topic:   getEnv("KAFKA_TOPIC", "notifications"),
			GroupID: getEnv("KAFKA_GROUP_ID", "notification-consumers"),
		},
	}

	if config.TelegramConfig.TokenWriteBot == "" {
		return nil, fmt.Errorf("TOKEN_WRITE_BOT is required")
	}

	if config.TelegramConfig.TokenNotifyBot == "" {
		return nil, fmt.Errorf("TOKEN_NOTIFY_BOT is required")
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
