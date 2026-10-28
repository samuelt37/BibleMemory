package notes

import "time"

type Note struct {
	ID         int       `json:"id"`
	UserID     int       `json:"-"`
	Filename   string    `json:"filename"`
	SourceType string    `json:"sourceType"` // "file" | "text"
	R2Key      *string   `json:"-"`
	MimeType   *string   `json:"mimeType,omitempty"`
	FileSize   *int64    `json:"fileSize,omitempty"`
	RawText    *string   `json:"-"` // not sent to frontend in list view; only via a dedicated "get content" endpoint if needed
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type Chunk struct {
	ID      int        `json:"id"`
	NoteID  int        `json:"noteId"`
	UserID  int        `json:"-"`
	Content string     `json:"content"`
	Refs    []verseRef `json:"refs"`
}

type NoteChunk = Chunk

type ChunkHit struct {
	ChunkID int
	NoteID  int
	Title   string
	Content string
}

type NoteFilter struct {
	Query   string
	BookID  *int
	Chapter *int
}

type verseRef struct {
	ID         *int    `json:"id,omitempty"`
	BookID     *int    `json:"bookId"`
	EndBookID  *int    `json:"endBookId,omitempty"`
	Chapter    *int    `json:"chapter,omitempty"`
	EndChapter *int    `json:"endChapter,omitempty"`
	VerseStart *int    `json:"verseStart,omitempty"`
	VerseEnd   *int    `json:"verseEnd,omitempty"`
	Source     *string `json:"source,omitempty"`
}

type chunkMatch struct {
	Refs []verseRef `json:"refs"`
}

type chunkResult struct {
	content   string
	match     *chunkMatch
	embedding []float32
	err       error
}

type DeletedRef struct {
	ChunkID int `json:"chunkId"`
	verseRef
}

type CreateChunkInput struct {
	Content string     `json:"content"`
	Refs    []verseRef `json:"refs"`
}

type BookNoteCount struct {
	BookID int
	Count  int
}
