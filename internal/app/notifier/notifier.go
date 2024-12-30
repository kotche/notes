package notifier

import (
	"context"
	"errors"
	"fmt"
	"github.com/kotche/bot/internal/service/notes"
	"github.com/segmentio/kafka-go"
	"gopkg.in/telebot.v3"
	"log"
	"time"
)

const (
	longProcessTimeout = 2
	checkInterval      = 10 * time.Second
	kafkaTopic         = "notifications"
	kafkaBroker        = "localhost:9092"
)

type Notifier struct {
	bot      *telebot.Bot
	notes    notes.Service
	producer *kafka.Writer
}

func New(bot *telebot.Bot, notes notes.Service) *Notifier {
	// TODO(cheki) Инициализация Kafka producer (вынести отдельно)
	producer := &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
		Topic:    kafkaTopic,
		Balancer: &kafka.LeastBytes{},
	}

	return &Notifier{bot: bot, notes: notes, producer: producer}
}

func (w *Notifier) Start() {
	log.Println("Notifier started...")

	if err := w.sendNotifications(); err != nil {
		log.Printf("error sending notifications: %v", err)
	}

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := w.sendNotifications(); err != nil {
			log.Printf("error sending notifications: %v", err)
		}
	}
}

func (w *Notifier) sendNotifications() error {
	ctx, cancel := context.WithTimeout(context.Background(), longProcessTimeout*time.Second)
	defer cancel()

	start := time.Now().Truncate(checkInterval)
	end := start.Add(checkInterval)

	log.Printf("ReceiveNotifications start '%s' and end '%s'", start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"))

	notifications, err := w.notes.ReceiveNotifications(ctx, start, end)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("context deadline exceeded while receive notifications: %v", err)
		}
		return err
	}

	for _, note := range notifications {
		message := fmt.Sprintf("%s (id %d)", note.Text, note.ID)

		if _, err = w.bot.Send(&telebot.User{ID: int64(note.UserID)}, message); err != nil {
			return fmt.Errorf("failed to send notification to user %d: %v", note.UserID, err)
		} else {
			log.Printf("notification sent to user %d: %s", note.UserID, message)
		}

		// Отправляем note.ID в очередь Kafka
		if err = w.producer.WriteMessages(ctx, kafka.Message{
			Key:   []byte(fmt.Sprintf("user-%d", note.UserID)),
			Value: []byte(fmt.Sprintf("%d", note.ID)),
		}); err != nil {
			log.Printf("failed to write message to Kafka: %v", err)
		} else {
			log.Printf("notification sent to user %d: %s, note.ID sent to Kafka", note.UserID, message)
		}
	}

	return nil
}
