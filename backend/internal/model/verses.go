package model

type Verse struct {
	Book string `json:"book"`
	Chapter string `json:"chapter"`
    Verse string `json:"verse"`
    Text  string `json:"text"`
}
