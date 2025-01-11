package notes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/kotche/bot/infrastructure/tracing"
	"github.com/kotche/bot/internal/model"
	_ "github.com/lib/pq"
	"time"

	"github.com/Masterminds/squirrel"
)

type DefaultRepository struct {
	db *sql.DB
}

func NewDefaultRepository(pg *sql.DB) *DefaultRepository {
	return &DefaultRepository{pg}
}

func (d *DefaultRepository) UserExists(ctx context.Context, userID model.UserID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND deleted_at IS NULL)`
	err := d.db.QueryRowContext(ctx, query, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to get user '%d' exists: %w", userID, err)
	}
	return exists, nil
}

func (d *DefaultRepository) CreateUser(ctx context.Context, user model.User) error {
	query := `INSERT INTO users (id, login, created_at) VALUES ($1, $2, NOW())`
	if _, err := d.db.ExecContext(ctx, query, user.ID, user.Login); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (d *DefaultRepository) CreateNote(ctx context.Context, note model.Note) (model.NoteID, error) {
	query := `
		INSERT INTO notes (user_id, text, notify_at, created_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id
	`

	var noteID model.NoteID
	err := d.db.QueryRowContext(ctx, query, note.UserID, note.Text, note.NotifyAt).Scan(&noteID)
	if err != nil {
		return 0, fmt.Errorf("failed to create note: %w", err)
	}

	return noteID, nil
}

func (d *DefaultRepository) NoteExists(ctx context.Context, noteID model.NoteID, userID model.UserID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM notes WHERE id = $1 AND user_id = $2)`
	err := d.db.QueryRowContext(ctx, query, noteID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to get note '%d' for user '%d' exists: %w", noteID, userID, err)
	}
	return exists, nil
}

func (d *DefaultRepository) GetNote(ctx context.Context, noteID model.NoteID, userID model.UserID) (*model.Note, error) {
	note := &model.Note{}
	query := `SELECT id, user_id, text, notify_at, created_at, deleted_at FROM notes WHERE id = $1 AND user_id = $2`
	err := d.db.QueryRowContext(ctx, query, noteID, userID).Scan(&note.ID, &note.UserID, &note.Text, &note.NotifyAt, &note.CreatedAt, &note.DeletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNoteNotFound
		}
		return nil, fmt.Errorf("failed to get note '%d' for user '%d': %w", noteID, userID, err)
	}
	return note, nil
}

func (d *DefaultRepository) DeleteNote(ctx context.Context, noteID model.NoteID, userID model.UserID) error {
	query := `
		UPDATE notes SET deleted_at = NOW() WHERE id = $1 AND user_id = $2
	`

	if _, err := d.db.ExecContext(ctx, query, noteID, userID); err != nil {
		return fmt.Errorf("failed to delete note %d for user %d: %w", noteID, userID, err)
	}

	return nil
}

func (d *DefaultRepository) ListNotes(ctx context.Context, userID model.UserID, showDeleted bool) ([]model.Note, error) {
	ctx, span := tracing.StartSpan(ctx, "ListNotes_repo")
	defer span.End()

	queryBuilder := squirrel.
		Select("id",
			"text",
			"notify_at",
			"created_at",
			"deleted_at").
		From("notes").
		Where(squirrel.Eq{"user_id": userID})

	if !showDeleted {
		queryBuilder = queryBuilder.Where("deleted_at IS NULL")
	}

	queryBuilder = queryBuilder.OrderBy("deleted_at DESC, notify_at").
		PlaceholderFormat(squirrel.Dollar)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query notes: %w", err)
	}
	defer rows.Close()

	var notes []model.Note
	for rows.Next() {
		var note model.Note
		if err = rows.Scan(&note.ID, &note.Text, &note.NotifyAt, &note.CreatedAt, &note.DeletedAt); err != nil {
			return nil, fmt.Errorf("failed to scan note: %w", err)
		}
		notes = append(notes, note)
	}

	return notes, nil
}

func (d *DefaultRepository) ReceiveNotifications(ctx context.Context, startTime, endTime time.Time) ([]model.Note, error) {
	queryBuilder := squirrel.
		Select("id",
			"user_id",
			"text",
			"notify_at",
			"created_at").
		From("notes").
		Where("deleted_at IS NULL").
		Where("notify_at >= ? AND notify_at < ?", startTime, endTime).
		OrderBy("notify_at").
		PlaceholderFormat(squirrel.Dollar)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query receive notifications: %w", err)
	}
	defer rows.Close()

	var notes []model.Note
	for rows.Next() {
		var note model.Note
		if err = rows.Scan(&note.ID, &note.UserID, &note.Text, &note.NotifyAt, &note.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan note: %w", err)
		}
		notes = append(notes, note)
	}

	return notes, nil
}
