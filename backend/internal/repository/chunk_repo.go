package repository

import (
	"database/sql"

	"github.com/pgvector/pgvector-go"
	"github.com/samuelt37/BibleMemory/internal/model"
)

type NoteChunkRepository struct {
	db *sql.DB
}

func NewNoteChunkRepository(db *sql.DB) *NoteChunkRepository {
	return &NoteChunkRepository{db: db}
}

func (r *NoteChunkRepository) Create(userID, noteID int, content string, embedding []float32, bookID, chapter, verseStart, verseEnd *int) error {
	_, err := r.db.Exec(
		`INSERT INTO note_chunks (note_id, user_id, content, embedding, book_id, chapter, verse_start, verse_end)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		noteID, userID, content, pgvector.NewVector(embedding), bookID, chapter, verseStart, verseEnd,
	)
	return err
}

func (r *NoteChunkRepository) ListByNote(noteID int) ([]model.NoteChunk, error) {
	rows, err := r.db.Query(
		`SELECT id, note_id, content, book_id, chapter, verse_start, verse_end, confirmed
		 FROM note_chunks WHERE note_id = $1`,
		noteID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []model.NoteChunk
	for rows.Next() {
		var c model.NoteChunk
		if err := rows.Scan(&c.ID, &c.NoteID, &c.Content, &c.BookID, &c.Chapter, &c.VerseStart, &c.VerseEnd, &c.Confirmed); err != nil {
			return nil, err
		}
		chunks = append(chunks, c)
	}
	return chunks, rows.Err()
}

func (r *NoteChunkRepository) FindRelevant(userID, startBookID, endBookID int) ([]string, error) {
	loBook, hiBook := startBookID, endBookID
	if hiBook < loBook {
		loBook, hiBook = hiBook, loBook
	}

	rows, err := r.db.Query(
		`SELECT content FROM note_chunks
		 WHERE user_id = $1 AND book_id BETWEEN $2 AND $3
		 ORDER BY created_at DESC
		 LIMIT 5`,
		userID, loBook, hiBook,
	)
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
