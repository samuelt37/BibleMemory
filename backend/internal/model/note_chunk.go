package model

type NoteChunk struct {
	ID         int    `json:"id"`
	NoteID     int    `json:"noteId"`
	UserID     int    `json:"-"`
	Content    string `json:"content"`
	BookID     *int   `json:"bookId"`
	Chapter    *int   `json:"chapter"`
	VerseStart *int   `json:"verseStart"`
	VerseEnd   *int   `json:"verseEnd"`
	Confirmed  bool   `json:"confirmed"`
}
