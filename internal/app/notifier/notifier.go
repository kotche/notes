package notifier

import (
	"context"
	"fmt"
	"github.com/kotche/bot/internal/model"
	"github.com/kotche/bot/internal/service/kafka"
	"github.com/kotche/bot/internal/service/notes"
	"gopkg.in/telebot.v3"
	"log"
	"strconv"
	"time"
)

const (
	checkInterval = time.Minute
)

type Notifier struct {
	bot    *telebot.Bot
	notes  notes.Service
	broker kafka.MessageBroker
}

func New(bot *telebot.Bot, notes notes.Service, broker kafka.MessageBroker) *Notifier {
	return &Notifier{
		bot:    bot,
		notes:  notes,
		broker: broker,
	}
}

func (n *Notifier) Start() {
	log.Println("Notifier started...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := n.sendNotifications(ctx); err != nil {
		log.Printf("error sending notifications: %v", err)
	}

	go func() {
		if err := n.runDeleteSentNotes(ctx); err != nil {
			log.Printf("error deleting sent notes: %v", err)
		}
	}()

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := n.sendNotifications(ctx); err != nil {
			log.Printf("error sending notifications: %v", err)
		}
	}
}

func (n *Notifier) sendNotifications(ctx context.Context) error {
	//start := time.Date(2025, 1, 8, 15, 30, 0, 0, time.Local)
	start := time.Now().Truncate(checkInterval)
	end := start.Add(checkInterval)

	//log.Printf("ReceiveNotifications start '%s' and end '%s'", start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"))

	notifications, err := n.notes.ReceiveNotifications(ctx, start, end)
	if err != nil {
		return err
	}

	for _, note := range notifications {
		message := fmt.Sprintf("%s (id %d)", note.Text, note.ID)

		if _, err = n.bot.Send(&telebot.User{ID: int64(note.UserID)}, message); err != nil {
			return fmt.Errorf("failed to send notification to user %d: %v", note.UserID, err)
		} else {
			log.Printf("notification sent to user %d: %s", note.UserID, message)
		}

		if err = n.broker.SendMessage(ctx,
			[]byte(fmt.Sprintf("%d", note.UserID)),
			[]byte(fmt.Sprintf("%d", note.ID)),
		); err != nil {
			log.Printf("failed to send message to kafka: %v", err)
		} else {
			log.Printf("note '%d' for user '%d' sent to kafka", note.ID, note.UserID)
		}
	}

	return nil
}

func (n *Notifier) runDeleteSentNotes(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		key, val, err := n.broker.ReadMessage(ctx)
		if err != nil {
			log.Printf("error reading message from kafka: %v", err)
			continue
		}

		userID, err := strconv.ParseInt(string(key), 10, 64)
		if err != nil {
			log.Printf("error converting user id `%s` to int: %v", key, err)
			continue
		}

		noteID, err := strconv.ParseInt(string(val), 10, 64)
		if err != nil {
			log.Printf("error converting note id `%s` to int: %v", key, err)
			continue
		}

		//TODO(cheki) удалять батчами
		if err = n.notes.Delete(ctx, model.NoteID(noteID), model.UserID(userID)); err != nil {
			log.Printf("error deleting note %d: %v", noteID, err)
		}

		log.Printf("note '%d' for user '%d' deleted", noteID, userID)
	}
}
