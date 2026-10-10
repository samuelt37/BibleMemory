package notes

import (
	"database/sql"
	"encoding/json"

	"github.com/lib/pq"
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

func (r *Repository) ListByUser(userID int, f NoteFilter) ([]Note, error) {
	rows, err := r.db.Query(
		`SELECT n.id, n.user_id, n.filename, n.source_type, n.r2_key, n.mime_type, n.file_size, n.status, n.created_at, n.updated_at
		FROM notes n
		WHERE n.user_id = $1
		AND ($2::text = '' OR n.filename ILIKE '%' || $2 || '%' OR n.raw_text ILIKE '%' || $2 || '%')
		AND ($3::int IS NULL OR EXISTS (
				SELECT 1 FROM chunk_refs cr
				WHERE cr.note_id = n.id AND cr.book_id = $3
				AND ($4::int IS NULL OR cr.chapter = $4)))
		ORDER BY n.created_at DESC`,
		userID, f.Query, f.BookID, f.Chapter,
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

func (r *Repository) ListByIDs(userID int, ids []int) ([]Note, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.db.Query(
		`SELECT id, user_id, filename, source_type, r2_key, mime_type, file_size, status, created_at, updated_at
		 FROM notes WHERE user_id = $1 AND id = ANY($2)`,
		userID, pq.Array(ids),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.UserID, &n.Filename, &n.SourceType, &n.R2Key, &n.MimeType, &n.FileSize, &n.Status, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

type ChunkRepository struct {
	db *sql.DB
}

func NewChunkRepository(db *sql.DB) *ChunkRepository {
	return &ChunkRepository{db: db}
}

func (r *ChunkRepository) Create(userID, noteID int, content string, embedding []float32, refs []verseRef) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // no-op after a successful Commit

	var chunkID int
	err = tx.QueryRow(
		`INSERT INTO note_chunks (note_id, user_id, content, embedding)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		noteID, userID, content, pgvector.NewVector(embedding),
	).Scan(&chunkID)
	if err != nil {
		return err
	}

	// only tagged chunks get a ref row
	for _, ref := range refs {
		if ref.BookID == nil {
			continue
		}
		_, err = tx.Exec(
			`INSERT INTO chunk_refs (user_id, note_id, chunk_id, book_id, chapter, verse_start, verse_end)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 ON CONFLICT DO NOTHING`,
			userID, noteID, chunkID, *ref.BookID, ref.Chapter, ref.VerseStart, ref.VerseEnd,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *ChunkRepository) ListByNote(noteID int) ([]Chunk, error) {
	rows, err := r.db.Query(
		`SELECT c.id, c.note_id, c.content, c.confirmed,
		        COALESCE(
		          json_agg(json_build_object(
		            'id', cr.id, 'bookId', cr.book_id, 'chapter', cr.chapter,
		            'verseStart', cr.verse_start, 'verseEnd', cr.verse_end,
		            'source', cr.source
		          ) ORDER BY cr.id) FILTER (WHERE cr.id IS NOT NULL),
		          '[]'
		        )
		 FROM note_chunks c
		 LEFT JOIN chunk_refs cr ON cr.chunk_id = c.id
		 WHERE c.note_id = $1
		 GROUP BY c.id
		 ORDER BY c.id`,
		noteID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chunks := []Chunk{}
	for rows.Next() {
		var c Chunk
		var raw []byte
		if err := rows.Scan(&c.ID, &c.NoteID, &c.Content, &c.Confirmed, &raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &c.Refs); err != nil {
			return nil, err
		}
		chunks = append(chunks, c)
	}
	return chunks, rows.Err()
}

func (r *ChunkRepository) FindReferences(userID, startBookID, endBookID int, queryEmb []float32) ([]ChunkHit, error) {
	loBook, hiBook := startBookID, endBookID
	if hiBook < loBook {
		loBook, hiBook = hiBook, loBook
	}

	query := `SELECT c.id, c.note_id, n.filename, c.content
			FROM note_chunks c
			JOIN notes n ON n.id = c.note_id
			WHERE c.user_id = $1
			AND EXISTS (SELECT 1 FROM chunk_refs cr
						WHERE cr.chunk_id = c.id AND cr.book_id BETWEEN $2 AND $3)
			ORDER BY c.created_at DESC
			LIMIT 5`
	args := []any{userID, loBook, hiBook}

	if queryEmb != nil {
		query = `SELECT c.id, c.note_id, n.filename, c.content
	         FROM note_chunks c
	         JOIN notes n ON n.id = c.note_id
	         WHERE c.user_id = $1 
			 AND EXISTS (SELECT 1 FROM chunk_refs cr
						WHERE cr.chunk_id = c.id AND cr.book_id BETWEEN $2 AND $3)
	         AND c.embedding IS NOT NULL
	         ORDER BY c.embedding <=> $4
	         LIMIT 5`
		args = append(args, pgvector.NewVector(queryEmb))
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contents []ChunkHit
	for rows.Next() {
		var c ChunkHit
		if err := rows.Scan(&c.ChunkID, &c.NoteID, &c.Title, &c.Content); err != nil {
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
		`SELECT count(*) 
		FROM note_chunks c
		WHERE user_id = $1 
		AND EXISTS (SELECT 1 FROM chunk_refs cr
						WHERE cr.chunk_id = c.id AND cr.book_id BETWEEN $2 AND $3)`,
		userID, loBook, hiBook,
	).Scan(&count)
	return count, err
}

func (r *ChunkRepository) SearchNoteIDs(
	userID int,
	queryEmb []float32,
	bookID, chapter *int,
	maxDist float64,
	limit int,
) ([]int, error) {
	rows, err := r.db.Query(
		`SELECT c.note_id
		FROM note_chunks c
		WHERE c.user_id = $1
		AND c.embedding IS NOT NULL
		AND ($3::int IS NULL OR EXISTS (
				SELECT 1 FROM chunk_refs cr
				WHERE cr.chunk_id = c.id
				AND cr.book_id = $3
				AND ($4::int IS NULL OR cr.chapter = $4)))
		GROUP BY c.note_id
		HAVING MIN(c.embedding <=> $2) <= $5
		ORDER BY MIN(c.embedding <=> $2)
		LIMIT $6`,
		userID, pgvector.NewVector(queryEmb), bookID, chapter, maxDist, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *ChunkRepository) ReplaceRefs(userID, noteID, chunkID int, refs []verseRef) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec(`DELETE FROM chunk_refs WHERE chunk_id = $1 AND source = 'auto'`, chunkID); err != nil {
		return err
	}
	for _, ref := range refs {
		if ref.BookID == nil {
			continue
		}
		if _, err = tx.Exec(
			`INSERT INTO chunk_refs (user_id, note_id, chunk_id, book_id, chapter, verse_start, verse_end)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 ON CONFLICT DO NOTHING`,
			userID, noteID, chunkID, *ref.BookID, ref.Chapter, ref.VerseStart, ref.VerseEnd,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}
