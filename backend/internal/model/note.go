package model

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
