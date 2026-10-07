package summary

import "github.com/samuelt37/BibleMemory/internal/scripture"

type NoteUse struct {
	ID           int    `json:"id"`
	Quote        string `json:"quote"`
	SummaryQuote string `json:"summaryQuote,omitempty"`
	NoteQuote    string `json:"noteQuote"`
}

type Request struct {
	Scripture scripture.Query `json:"scripture"`
	Answers   []string        `json:"answers"`
}

type NoteRef struct {
	NoteID       int    `json:"noteId"`
	Title        string `json:"title"`
	Quote        string `json:"quote,omitempty"`
	SummaryQuote string `json:"summaryQuote"`
	NoteQuote    string `json:"noteQuote,omitempty"`
}

type Result struct {
	Accuracy  int       `json:"accuracy"`
	Feedback  string    `json:"feedback"`
	NotesUsed []NoteUse `json:"notesUsed,omitempty"` // from the model, internal
	Notes     []NoteRef `json:"notes,omitempty"`
}

type SummaryRequest = Request
type SummaryResult = Result
