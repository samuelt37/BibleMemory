package repository

import (
	"database/sql"

	"github.com/samuelt37/BibleMemory/internal/model"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) InitTable() error {
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

func (r *UserRepository) GetByClerkID(clerkUserID string) (*model.User, error) {
	var u model.User
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

func (r *UserRepository) Create(clerkUserID string) (*model.User, error) {
	var u model.User
	err := r.db.QueryRow(
		`INSERT INTO users (clerk_user_id) VALUES ($1) RETURNING id, clerk_user_id`,
		clerkUserID,
	).Scan(&u.ID, &u.ClerkUserID)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
