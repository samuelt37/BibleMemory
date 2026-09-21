package repository

import (
	"database/sql"

	"github.com/samuelt37/BibleMemory/internal/model"
)

type NoteRepository struct {
	db *sql.DB
}

func NewNoteRepository(db *sql.DB) *NoteRepository {
	return &NoteRepository{db: db}
}

func (r *NoteRepository) CreateNote(userID int, filename, r2Key, mimeType string, fileSize int64) (*model.Note, error) {
	var n model.Note
	err := r.db.QueryRow(
		`INSERT INTO notes (user_id, filename, r2_key, mime_type, file_size, status)
		 VALUES ($1, $2, $3, $4, $5, 'processing')
		 RETURNING id, user_id, filename, r2_key, mime_type, file_size, status, created_at, updated_at`,
		userID, filename, r2Key, mimeType, fileSize,
	).Scan(&n.ID, &n.UserID, &n.Filename, &n.R2Key, &n.MimeType, &n.FileSize, &n.Status, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *NoteRepository) CreateTextNote(userID int, filename, text string) (*model.Note, error) {
	var n model.Note
	err := r.db.QueryRow(
		`INSERT INTO notes (user_id, filename, source_type, raw_text, status)
		 VALUES ($1, $2, 'text', $3, 'processing')
		 RETURNING id, user_id, filename, source_type, r2_key, mime_type, file_size, raw_text, status, created_at, updated_at`,
		userID, filename, text,
	).Scan(&n.ID, &n.UserID, &n.Filename, &n.SourceType, &n.R2Key, &n.MimeType, &n.FileSize, &n.RawText, &n.Status, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *NoteRepository) ListByUser(userID int) ([]model.Note, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, filename, source_type, r2_key, mime_type, file_size, status, created_at, updated_at
		 FROM notes WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []model.Note
	for rows.Next() {
		var n model.Note
		if err := rows.Scan(&n.ID, &n.UserID, &n.Filename, &n.SourceType, &n.R2Key, &n.MimeType, &n.FileSize, &n.Status, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func (r *NoteRepository) GetByID(userID, noteID int) (*model.Note, error) {
	var n model.Note
	err := r.db.QueryRow(
		`SELECT id, user_id, filename, source_type, r2_key, mime_type, file_size, raw_text, status, created_at, updated_at
		 FROM notes WHERE id = $1 AND user_id = $2`,
		noteID, userID,
	).Scan(&n.ID, &n.UserID, &n.Filename, &n.SourceType, &n.R2Key, &n.MimeType, &n.FileSize, &n.RawText, &n.Status, &n.CreatedAt, &n.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *NoteRepository) UpdateStatus(noteID int, status string) error {
	_, err := r.db.Exec(
		`UPDATE notes SET status = $1, updated_at = now() WHERE id = $2`,
		status, noteID,
	)
	return err
}

func (r *NoteRepository) Delete(userID, noteID int) error {
	_, err := r.db.Exec(`DELETE FROM notes WHERE id = $1 AND user_id = $2`, noteID, userID)
	return err
}
