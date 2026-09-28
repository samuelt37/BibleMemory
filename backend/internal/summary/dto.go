package summary

import "github.com/samuelt37/BibleMemory/internal/scripture"

type Request struct {
	Scripture scripture.Query `json:"scripture"`
	Answers   []string        `json:"answers"`
}

type Result struct {
	Accuracy int    `json:"accuracy"`
	Feedback string `json:"feedback"`
}

type SummaryRequest = Request
type SummaryResult = Result
