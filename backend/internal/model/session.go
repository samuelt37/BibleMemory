package model

import "time"

type ScriptureRange struct {
	StartBookID  int  `json:"startBookId"`
	StartChapter int  `json:"startChapter"`
	StartVerse   *int `json:"startVerse"`
	EndBookID    int  `json:"endBookId"`
	EndChapter   int  `json:"endChapter"`
	EndVerse     *int `json:"endVerse"`
}

type MemorySession struct {
	ID         int              `json:"id"`
	UserID     int              `json:"-"`
	Ranges     []ScriptureRange `json:"ranges"`
	Bookmarked bool             `json:"bookmarked"`
	CreatedAt  time.Time        `json:"createdAt"`
}
