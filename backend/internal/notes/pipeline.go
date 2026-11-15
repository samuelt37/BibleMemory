package notes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/samuelt37/BibleMemory/internal/gemini"
	"golang.org/x/sync/errgroup"
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

const videoWindowSec = 600 // 10 minutes

const videoPrompt = `Transcribe every word spoken in this video's audio track, verbatim. Do not summarize, condense, or skip any speech.

TIMESTAMPS
- Start a new paragraph about every 30-60 seconds, and begin each paragraph with the time it starts as [MM:SS]. Use [H:MM:SS] once past one hour.
- Timestamps are relative to the start of THIS clip. The clip begins at [0:00], no matter where it sits in the original video. Never use the original video's time.
- Put a timestamp only at the start of a paragraph, never mid-sentence and never on every line.

SLIDES
- When a slide or on-screen text changes to something new, add its text on its own line in the form [Slide: ...] at that point in the transcript.
- Do not repeat a slide that is still on screen.
- Ignore countdown timers, clocks, progress bars, and placeholder screens such as "please prepare your hearts for worship".

PRIORITIES
- Speech comes first: never replace spoken words with slide text, and do not describe images.
- If nobody is speaking during a stretch, return nothing for it.

Return ONLY the transcript, with no other commentary or headers.`

const videoParallelism = 3

func extractTextFromVideo(ctx context.Context, client *gemini.Client, f *gemini.File) (string, error) {
	dur := f.DurationSeconds()
	if dur <= 0 {
		log.Println("extractTextFromVideo: no duration reported, transcribing in one pass")
		return transcribeWindow(ctx, client, f, 0, 0)
	}

	type win struct{ start, end int }
	var wins []win
	for start := 0; float64(start) < dur; start += videoWindowSec {
		end := start + videoWindowSec
		if float64(end) > dur {
			end = int(math.Ceil(dur))
		}
		wins = append(wins, win{start, end})
	}

	parts := make([]string, len(wins))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(videoParallelism)
	for i, w := range wins {
		i, w := i, w
		g.Go(func() error {
			part, err := withRetry(gctx, 4, func() (string, error) {
				return transcribeWindow(gctx, client, f, w.start, w.end)
			})
			if err != nil {
				return fmt.Errorf("window %d-%d: %w", w.start, w.end, err)
			}
			parts[i] = strings.TrimSpace(part)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return "", err
	}

	var sb strings.Builder
	for i, w := range wins {
		part := parts[i]
		offset := w.start
		// Model sometimes returns absolute time instead of clip-relative.
		if m := anyTsRe.FindStringSubmatchIndex(part); m != nil && w.start > 0 {
			if float64(idxSeconds(part, m)) >= float64(w.start)-5 {
				offset = 0
			}
		}
		log.Printf("extractTextFromVideo: window %ds offset=%d first line: %.60s", w.start, offset, part)
		sb.WriteString(strings.TrimSpace(offsetTimestamps(part, offset)))
		sb.WriteString("\n\n")
	}
	return sb.String(), nil
}

func withRetry(ctx context.Context, attempts int, fn func() (string, error)) (string, error) {
	var lastErr error
	for i := 0; i < attempts; i++ {
		out, err := fn()
		if err == nil {
			return out, nil
		}
		lastErr = err
		backoff := time.Duration(1<<i) * time.Second
		backoff += time.Duration(rand.Intn(1000)) * time.Millisecond // jitter
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(backoff):
		}
	}
	return "", lastErr
}

func transcribeWindow(ctx context.Context, client *gemini.Client, f *gemini.File, start, end int) (string, error) {
	filePart := map[string]any{
		"file_data": map[string]string{"mime_type": f.MimeType, "file_uri": f.URI},
	}
	prompt := videoPrompt

	if end > 0 {
		filePart["video_metadata"] = map[string]string{
			"start_offset": fmt.Sprintf("%ds", start),
			"end_offset":   fmt.Sprintf("%ds", end),
		}
		prompt = fmt.Sprintf("This clip covers %d:%02d to %d:%02d of the video. ", start/60, start%60, end/60, end%60) + prompt
	}

	body := map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]any{filePart, {"text": prompt}}},
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
		"generationConfig": map[string]any{
			"responseMimeType": "application/json", // forces valid JSON, no fences or chatter
			"temperature":      0,
			"maxOutputTokens":  4096,
		},
	}

	rawText, err := client.GenerateContent(ctx, body)
	if err != nil {
		return nil, err
	}

	raw := cleanJSONFence(rawText)
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("failed to parse match response: empty response (chunk len %d)", len(chunk))
	}

	retryTemp := 0.7
	attempts := []*float64{nil, &retryTemp} // first: default, second: slightly hotter

	var match chunkMatch
	var parseErr error
	for _, temp := range attempts {
		actx, cancel := context.WithTimeout(ctx, 30*time.Second)
		rawText, err := client.GenerateContent(actx, buildMatchBody(prompt, temp))
		cancel()
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
				parseErr = err // runaway generation: give the second attempt its own 30s
				continue
			}
			return nil, err
		}
		raw := cleanJSONFence(rawText)
		if strings.TrimSpace(raw) == "" {
			parseErr = fmt.Errorf("empty response (chunk len %d)", len(chunk))
			continue
		}
		match = chunkMatch{}
		if parseErr = json.Unmarshal([]byte(raw), &match); parseErr == nil {
			break
		}
		logBadJSON(raw, parseErr, len(chunk))
	}
	if parseErr != nil {
		return nil, fmt.Errorf("failed to parse match response: %w", parseErr)
	}
	match.Refs = dedupeOverlaps(match.Refs)
	for i := range match.Refs {
		match.Refs[i].Quotes = validQuotes(chunk, match.Refs[i].Quotes)
	}
	return &match, nil
}

var matchSchema = func() map[string]any {
	nullInt := map[string]any{"type": "INTEGER", "nullable": true}
	return map[string]any{
		"type": "OBJECT",
		"properties": map[string]any{
			"refs": map[string]any{
				"type": "ARRAY",
				"items": map[string]any{
					"type": "OBJECT",
					"properties": map[string]any{
						"bookId":     map[string]any{"type": "INTEGER"},
						"chapter":    nullInt,
						"verseStart": nullInt,
						"verseEnd":   nullInt,
						"quotes": map[string]any{
							"type":  "ARRAY",
							"items": map[string]any{"type": "STRING"},
						},
					},
					"required": []string{"bookId", "quotes"},
				},
			},
		},
		"required": []string{"refs"},
	}
}()

func buildMatchBody(prompt string, temp *float64) map[string]any {
	cfg := map[string]any{
		"responseMimeType": "application/json",
		"responseSchema":   matchSchema,
		"maxOutputTokens":  1500, // was 4096; a valid response is a few hundred tokens
	}
	if temp != nil {
		cfg["temperature"] = *temp // only set on the retry; nil = model default
	}
	return map[string]any{
		"contents":         []map[string]any{{"parts": []map[string]string{{"text": prompt}}}},
		"generationConfig": cfg,
	}
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

func logBadJSON(raw string, err error, chunkLen int) {
	var se *json.SyntaxError
	if errors.As(err, &se) {
		off := int(se.Offset)
		lo, hi := max(0, off-80), min(len(raw), off+80)
		log.Printf("matchChunkToVerse: bad JSON at byte %d (chunk len %d): ...%s...", off, chunkLen, raw[lo:hi])
		return
	}
	log.Printf("matchChunkToVerse: bad JSON (chunk len %d): %.300s", chunkLen, raw)
}
