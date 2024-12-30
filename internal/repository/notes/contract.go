package notes

import (
	"context"
	"github.com/kotche/bot/internal/model"
	"time"
)

type (
	Repository interface {
		UserExists(ctx context.Context, userID model.UserID) (bool, error)
		CreateUser(ctx context.Context, user model.User) error
		CreateNote(ctx context.Context, note model.Note) (model.NoteID, error)
		NoteExists(ctx context.Context, noteID model.NoteID, userID model.UserID) (bool, error)
		GetNote(ctx context.Context, noteID model.NoteID, userID model.UserID) (*model.Note, error)
		DeleteNote(ctx context.Context, noteID model.NoteID, userID model.UserID) error
		ListNotes(ctx context.Context, userID model.UserID, showDeleted bool) ([]model.Note, error)
		ReceiveNotifications(ctx context.Context, startTime, endTime time.Time) ([]model.Note, error)
	}
)
