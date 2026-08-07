package dto

type ScriptureQuery struct {
    Translation string
    Book        string
    Chapter     *int
    Verse       *int
    VerseStart  *int
    VerseEnd    *int
}
