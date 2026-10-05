package notes

import (
	"database/sql"

	"github.com/pgvector/pgvector-go"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateNote(userID int, filename, r2Key, mimeType string, fileSize int64) (*Note, error) {
	var n Note
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

func (r *Repository) CreateTextNote(userID int, filename, text string) (*Note, error) {
	var n Note
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

func (r *Repository) ListByUser(userID int) ([]Note, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, filename, source_type, r2_key, mime_type, file_size, status, created_at, updated_at
		 FROM notes WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notesList []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.UserID, &n.Filename, &n.SourceType, &n.R2Key, &n.MimeType, &n.FileSize, &n.Status, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		notesList = append(notesList, n)
	}
	return notesList, rows.Err()
}

func (r *Repository) GetByID(userID, noteID int) (*Note, error) {
	var n Note
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

func (r *Repository) UpdateStatus(noteID int, status string) error {
	_, err := r.db.Exec(
		`UPDATE notes SET status = $1, updated_at = now() WHERE id = $2`,
		status, noteID,
	)
	return err
}

func (r *Repository) Delete(userID, noteID int) error {
	_, err := r.db.Exec(`DELETE FROM notes WHERE id = $1 AND user_id = $2`, noteID, userID)
	return err
}

func (r *Repository) UpdateRawText(noteID int, text string) error {
	_, err := r.db.Exec(
		`UPDATE notes SET raw_text = $1, updated_at = now() WHERE id = $2`,
		text, noteID,
	)
	return err
}

type ChunkRepository struct {
	db *sql.DB
}

func NewChunkRepository(db *sql.DB) *ChunkRepository {
	return &ChunkRepository{db: db}
}

func (r *ChunkRepository) Create(userID, noteID int, content string, embedding []float32, bookID, chapter, verseStart, verseEnd *int) error {
	_, err := r.db.Exec(
		`INSERT INTO note_chunks (note_id, user_id, content, embedding, book_id, chapter, verse_start, verse_end)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		noteID, userID, content, pgvector.NewVector(embedding), bookID, chapter, verseStart, verseEnd,
	)
	return err
}

func (r *ChunkRepository) ListByNote(noteID int) ([]Chunk, error) {
	rows, err := r.db.Query(
		`SELECT id, note_id, content, book_id, chapter, verse_start, verse_end, confirmed
		 FROM note_chunks WHERE note_id = $1`,
		noteID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []Chunk
	for rows.Next() {
		var c Chunk
		if err := rows.Scan(&c.ID, &c.NoteID, &c.Content, &c.BookID, &c.Chapter, &c.VerseStart, &c.VerseEnd, &c.Confirmed); err != nil {
			return nil, err
		}
		chunks = append(chunks, c)
	}
	return chunks, rows.Err()
}

func (r *ChunkRepository) FindRelevant(userID, startBookID, endBookID int, queryEmb []float32) ([]string, error) {
	loBook, hiBook := startBookID, endBookID
	if hiBook < loBook {
		loBook, hiBook = hiBook, loBook
	}

	query := `SELECT content FROM note_chunks
	          WHERE user_id = $1 AND book_id BETWEEN $2 AND $3
	          ORDER BY created_at DESC
	          LIMIT 5`
	args := []any{userID, loBook, hiBook}

	if queryEmb != nil {
		query = `SELECT content FROM note_chunks
		         WHERE user_id = $1 AND book_id BETWEEN $2 AND $3
		           AND embedding IS NOT NULL
		         ORDER BY embedding <=> $4
		         LIMIT 5`
		args = append(args, pgvector.NewVector(queryEmb))
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contents []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		contents = append(contents, c)
	}
	return contents, rows.Err()
}

func (r *ChunkRepository) CountByBooks(userID, startBookID, endBookID int) (int, error) {
	loBook, hiBook := startBookID, endBookID
	if hiBook < loBook {
		loBook, hiBook = hiBook, loBook
	}

	var count int
	err := r.db.QueryRow(
		`SELECT count(*) FROM note_chunks WHERE user_id = $1 AND book_id BETWEEN $2 AND $3`,
		userID, loBook, hiBook,
	).Scan(&count)
	return count, err
}

