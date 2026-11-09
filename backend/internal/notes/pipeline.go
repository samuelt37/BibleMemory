package notes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/samuelt37/BibleMemory/internal/gemini"
)

const transcribePrompt = `Transcribe all text in this file exactly as written, including handwriting.
Preserve line breaks and paragraph order. Mark unreadable words as [illegible].
Return ONLY the transcribed text, with no commentary.`

func extractTextFromFile(ctx context.Context, client *gemini.Client, fileBytes []byte, mimeType string) (string, error) {
	encoded := base64.StdEncoding.EncodeToString(fileBytes)

	body := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]any{
					{"text": transcribePrompt},
					{
						"inline_data": map[string]string{
							"mime_type": mimeType,
							"data":      encoded,
						},
					},
				},
			},
		},
	}

	return client.GenerateContent(ctx, body)
}

func extractTextFromVideo(ctx context.Context, client *gemini.Client, f *gemini.File) (string, error) {
	body := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]any{
					{"file_data": map[string]string{
						"mime_type": f.MimeType,
						"file_uri":  f.URI,
					}},
					{"text": "Transcribe every word spoken in this video's audio track, verbatim. Ignore slides and on-screen text entirely. Do not summarize, condense, or skip anything. Return ONLY the spoken words, with no commentary, headers, or timestamps."},
				},
			},
		},
	}
	return client.GenerateContentLong(ctx, body)
}

func chunkText(text string, targetWords int) []string {
	paragraphs := strings.Split(strings.TrimSpace(text), "\n\n")

	var chunks []string
	var current []string
	wordCount := 0

	flush := func() {
		if len(current) > 0 {
			chunks = append(chunks, strings.Join(current, "\n\n"))
			current = nil
			wordCount = 0
		}
	}

	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		words := strings.Fields(p)

		if len(words) > targetWords {
			flush()
			for i := 0; i < len(words); i += targetWords {
				end := i + targetWords
				if end > len(words) {
					end = len(words)
				}
				chunks = append(chunks, strings.Join(words[i:end], " "))
			}
			continue
		}

		if wordCount+len(words) > targetWords && len(current) > 0 {
			flush()
		}

		current = append(current, p)
		wordCount += len(words)
	}

	flush()
	return chunks
}

func matchChunkToVerse(ctx context.Context, client *gemini.Client, chunk string, bookList string) (*chunkMatch, error) {
	prompt := fmt.Sprintf(
		`Given this note text, identify every Bible passage it discusses or clearly alludes to.
		Books (id:name): %s

		Note text:
		%s

		Respond ONLY with JSON in this shape:
		{"refs": [{"bookId": 19, "chapter": 110, "verseStart": 4, "verseEnd": 4, "quotes": ["Mentioned in Psalms 110:4 a priesthood according to his order."]}]}

		Rules:
		- One object per distinct passage, at most 5. Most chunks have one or two.
		- Include passages the text names, and also passages it clearly alludes to even when not named (e.g. a note about Esau selling his birthright may also point to Hebrews 12:16-17).
		- bookId is required. Use null for chapter/verseStart/verseEnd if the text only refers to a whole book or whole chapter.
		- Only set verseStart/verseEnd when chapter is set. For a single verse, set both to the same number.
		- If the text isn't clearly about a specific passage, return {"refs": []}.
		- quotes: if chapter AND verseStart are both set, quotes MUST contain at least one entry. Each entry is a span of the note text copied character for character (a complete clause or sentence of at least 5 words, never a bare name), and it must be the part of the note that points to this passage. Give one entry per distinct place the note discusses the passage, up to 5.
		- If chapter or verseStart is null (the note only refers to a whole book or whole chapter), set quotes to [].`,
		bookList, chunk,
	)

	body := map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": prompt}}},
		},
	}

	rawText, err := client.GenerateContent(ctx, body)
	if err != nil {
		return nil, err
	}

	raw := cleanJSONFence(rawText)

	var match chunkMatch
	if err := json.Unmarshal([]byte(raw), &match); err != nil {
		return nil, fmt.Errorf("failed to parse match response: %w", err)
	}
	match.Refs = dedupeOverlaps(match.Refs)
	for i := range match.Refs {
		match.Refs[i].Quotes = validQuotes(chunk, match.Refs[i].Quotes)
	}
	return &match, nil
}

func dedupeOverlaps(refs []verseRef) []verseRef {
	out := make([]verseRef, 0, len(refs))
	for _, r := range refs {
		merged := false
		for i, o := range out {
			if r.BookID != nil && o.BookID != nil && *r.BookID == *o.BookID &&
				intEq(r.Chapter, o.Chapter) && rangesOverlap(r, o) {
				if o.VerseStart == nil && r.VerseStart != nil {
					// r is more specific than o; replace o with r and preserve quotes
					r.Quotes = append(r.Quotes, o.Quotes...)
					out[i] = r
				} else {
					out[i].Quotes = append(out[i].Quotes, r.Quotes...)
				}
				merged = true
				break
			}
		}
		if !merged {
			out = append(out, r)
		}
	}
	return out
}

func intEq(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func rangesOverlap(a, b verseRef) bool {
	if a.VerseStart == nil || b.VerseStart == nil { // whole chapter overlaps everything in it
		return true
	}
	aEnd, bEnd := *a.VerseStart, *b.VerseStart
	if a.VerseEnd != nil {
		aEnd = *a.VerseEnd
	}
	if b.VerseEnd != nil {
		bEnd = *b.VerseEnd
	}
	return *a.VerseStart <= bEnd && *b.VerseStart <= aEnd
}

var quoteFixer = strings.NewReplacer("’", "'", "‘", "'", "“", `"`, "”", `"`, "–", "-", "—", "-")

func normalize(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(quoteFixer.Replace(s)), " "))
}

func validQuote(chunk, quote string) *string {
	q := strings.TrimSpace(quote)
	if q == "" || !strings.Contains(normalize(chunk), normalize(q)) {
		return nil
	}
	return &q
}

func validQuotes(chunk string, quotes []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, q := range quotes {
		v := validQuote(chunk, q)
		if v == nil || seen[normalize(*v)] {
			log.Printf("quote rejected: %q", q)
			continue
		}
		seen[normalize(*v)] = true
		out = append(out, *v)
	}
	return out
}

var defaultClient = gemini.NewClient()

// EmbedText generates a 768-dim vector embedding using the shared gemini client.
func EmbedText(ctx context.Context, text string) ([]float32, error) {
	return defaultClient.EmbedText(ctx, text, 768)
}

func cleanJSONFence(s string) string {
	s = trimPrefixSuffix(s, "```json", "```")
	s = trimPrefixSuffix(s, "```", "```")
	return s
}

func trimPrefixSuffix(s, prefix, suffix string) string {
	s = trimSpaceBoth(s)
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		s = s[len(prefix):]
	}
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		s = s[:len(s)-len(suffix)]
	}
	return trimSpaceBoth(s)
}

func trimSpaceBoth(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\n' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\n' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
