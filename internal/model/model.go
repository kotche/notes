package model

import "time"

type (
	User struct {
		ID    UserID
		Login string
	}

	Note struct {
		ID        NoteID
		UserID    UserID
		Text      string
		NotifyAt  time.Time
		CreatedAt time.Time
		DeletedAt *time.Time
	}
)
