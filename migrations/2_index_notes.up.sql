CREATE INDEX idx_notes_notify_at_deleted_at_ordered
    ON notes (notify_at ASC)
    INCLUDE (id, user_id, text, created_at)
    WHERE deleted_at IS NULL;
