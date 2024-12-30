package notes

import (
	"context"
	"github.com/kotche/bot/internal/model"
	"github.com/kotche/bot/internal/repository/notes"
	"time"
)

type DefaultService struct {
	repo notes.Repository
}

func NewDefaultService(repo notes.Repository) *DefaultService {
	return &DefaultService{repo: repo}
}

func (d *DefaultService) EnsureUserExists(ctx context.Context, user model.User) error {
	exists, err := d.repo.UserExists(ctx, user.ID)
	if err != nil {
		return err
	}

	if !exists {
		err = d.repo.CreateUser(ctx, model.User{
			ID:    user.ID,
			Login: user.Login,
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func (d *DefaultService) Create(ctx context.Context, note model.Note) (model.NoteID, error) {
	return d.repo.CreateNote(ctx, note)
}

func (d *DefaultService) Get(ctx context.Context, noteID model.NoteID, userID model.UserID) (*model.Note, error) {
	return d.repo.GetNote(ctx, noteID, userID)
}

func (d *DefaultService) Delete(ctx context.Context, noteID model.NoteID, userID model.UserID) error {
	exists, err := d.repo.NoteExists(ctx, noteID, userID)
	if err != nil {
		return err
	}

	if !exists {
		return model.ErrNoteNotFound
	}

	return d.repo.DeleteNote(ctx, noteID, userID)
}

func (d *DefaultService) List(ctx context.Context, userID model.UserID, showDeleted bool) ([]model.Note, error) {
	return d.repo.ListNotes(ctx, userID, showDeleted)
}

func (d *DefaultService) ReceiveNotifications(ctx context.Context, startTime, endTime time.Time) ([]model.Note, error) {
	return d.repo.ReceiveNotifications(ctx, startTime, endTime)
}
