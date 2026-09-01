package model

type BookInfo struct {
	ID       int    `json:"id"`
	Book     string `json:"book"`
	Chapters int    `json:"chapters"`
}
