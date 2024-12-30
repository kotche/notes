package notes

import (
	"context"
	"github.com/kotche/bot/internal/model"
	"time"
)

type (
	Service interface {
		EnsureUserExists(ctx context.Context, user model.User) error
		Create(ctx context.Context, note model.Note) (model.NoteID, error)
		Get(ctx context.Context, noteID model.NoteID, userID model.UserID) (*model.Note, error)
		Delete(ctx context.Context, noteID model.NoteID, userID model.UserID) error
		List(ctx context.Context, userID model.UserID, showDeleted bool) ([]model.Note, error)
		ReceiveNotifications(ctx context.Context, startTime, endTime time.Time) ([]model.Note, error)
	}
)
