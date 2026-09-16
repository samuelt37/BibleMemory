package repository

import (
	"database/sql"
	"encoding/json"

	"github.com/samuelt37/BibleMemory/internal/model"
)

type SessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(userID int, ranges []model.ScriptureRange, bookmarked bool) (*model.MemorySession, error) {
	rangesJSON, err := json.Marshal(ranges)
	if err != nil {
		return nil, err
	}

	var s model.MemorySession
	var rawRanges []byte
	err = r.db.QueryRow(
		`INSERT INTO memory_sessions (user_id, ranges, bookmarked)
		 VALUES ($1, $2, $3)
		 RETURNING id, user_id, ranges, bookmarked, created_at`,
		userID, rangesJSON, bookmarked,
	).Scan(&s.ID, &s.UserID, &rawRanges, &s.Bookmarked, &s.CreatedAt)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(rawRanges, &s.Ranges); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SessionRepository) ListHistory(userID int, limit int) ([]model.MemorySession, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, ranges, bookmarked, created_at
		 FROM memory_sessions
		 WHERE user_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessions(rows)
}

func (r *SessionRepository) ListBookmarked(userID int) ([]model.MemorySession, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, ranges, bookmarked, created_at
		 FROM memory_sessions
		 WHERE user_id = $1 AND bookmarked = true
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessions(rows)
}

func (r *SessionRepository) SetBookmarked(userID int, sessionID int, bookmarked bool) error {
	_, err := r.db.Exec(
		`UPDATE memory_sessions SET bookmarked = $1 WHERE id = $2 AND user_id = $3`,
		bookmarked, sessionID, userID,
	)
	return err
}

func (r *SessionRepository) Delete(userID int, sessionID int) error {
	_, err := r.db.Exec(
		`DELETE FROM memory_sessions WHERE id = $1 AND user_id = $2`,
		sessionID, userID,
	)
	return err
}

func scanSessions(rows *sql.Rows) ([]model.MemorySession, error) {
	var out []model.MemorySession
	for rows.Next() {
		var s model.MemorySession
		var rawRanges []byte
		if err := rows.Scan(&s.ID, &s.UserID, &rawRanges, &s.Bookmarked, &s.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(rawRanges, &s.Ranges); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
