package model

type VerseInfo struct {
	Book      string `json:"book"`
	Chapter   int    `json:"chapter"`
	Verse     int    `json:"verse"`
	Text      string `json:"text"`
	Testament string `json:"testament"`
	BookOrder int    `json:"book_order"`
}
