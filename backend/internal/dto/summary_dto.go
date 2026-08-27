package dto

type SummaryRequest struct {
	Scripture ScriptureQuery `json:"scripture"`
	Answers   []string       `json:"answers"`
}

type SummaryResult struct {
	Accuracy int    `json:"accuracy"`
	Feedback string `json:"feedback"`
}
