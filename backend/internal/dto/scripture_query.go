package dto

type ScriptureQuery struct {
	Translation string
	Ranges      []ScriptureRange
}

type ScriptureRange struct {
	Start Reference
	End   *Reference
}

type Reference struct {
	Book    int
	Chapter *int
	Verse   *int
}
