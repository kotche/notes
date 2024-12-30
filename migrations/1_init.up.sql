CREATE TABLE IF NOT EXISTS users (
                     id INT8 PRIMARY KEY,
                     login TEXT NOT NULL UNIQUE,
                     created_at TIMESTAMP NOT NULL DEFAULT NOW(),
                     deleted_at TIMESTAMP
            );

CREATE TABLE IF NOT EXISTS notes (
        id SERIAL PRIMARY KEY,
        user_id INT8 NOT NULL,
        text TEXT NOT NULL,
        notify_at TIMESTAMP NOT NULL,
        created_at TIMESTAMP NOT NULL DEFAULT NOW(),
        deleted_at TIMESTAMP,
        CONSTRAINT fk_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
    );