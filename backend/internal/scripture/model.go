package scripture

type BookInfo struct {
	ID       int    `json:"id"`
	Book     string `json:"book"`
	Chapters int    `json:"chapters"`
}

type VerseInfo struct {
	Book      string `json:"book"`
	Chapter   int    `json:"chapter"`
	Verse     int    `json:"verse"`
	Text      string `json:"text"`
	Testament string `json:"testament,omitempty"`
	BookOrder int    `json:"book_order,omitempty"`
}

type VerseRecord struct {
	Translation string
	Testament   string
	BookOrder   int
	Book        string
	Chapter     int
	Verse       int
	Text        string
}

type Query struct {
	Translation string  `json:"translation"`
	Ranges      []Range `json:"ranges"`
}

type Range struct {
	Start Reference  `json:"start"`
	End   *Reference `json:"end"`
}

type Reference struct {
	Book    int  `json:"book"`
	Chapter *int `json:"chapter"`
	Verse   *int `json:"verse"`
}
