package user

import (
	"database/sql"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) InitTable() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		clerk_user_id TEXT UNIQUE NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);
	`
	_, err := r.db.Exec(schema)
	return err
}

func (r *Repository) GetByClerkID(clerkUserID string) (*User, error) {
	var u User
	err := r.db.QueryRow(
		`SELECT id, clerk_user_id FROM users WHERE clerk_user_id = $1`,
		clerkUserID,
	).Scan(&u.ID, &u.ClerkUserID)

	if err == sql.ErrNoRows {
		return nil, nil // not found, not an error
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) Create(clerkUserID string) (*User, error) {
	var u User
	err := r.db.QueryRow(
		`INSERT INTO users (clerk_user_id) VALUES ($1) RETURNING id, clerk_user_id`,
		clerkUserID,
	).Scan(&u.ID, &u.ClerkUserID)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
