package importer

type Book struct {
    Book     string    `json:"book"`
    Chapters []Chapter `json:"chapters"`
}

type Chapter struct {
    Chapter string  `json:"chapter"`
    Verses  []Verse `json:"verses"`
}

type Verse struct {
    Verse string `json:"verse"`
    Text  string `json:"text"`
}
