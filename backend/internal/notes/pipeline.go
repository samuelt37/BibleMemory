package notes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/samuelt37/BibleMemory/internal/gemini"
)

func extractTextFromFile(ctx context.Context, client *gemini.Client, fileBytes []byte, mimeType string) (string, error) {
	encoded := base64.StdEncoding.EncodeToString(fileBytes)

	body := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]any{
					{"text": "Extract all readable text from this document. Return ONLY the raw extracted text, with no commentary, headers, or formatting notes."},
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

type chunkMatch struct {
	BookID     *int `json:"bookId"`
	Chapter    *int `json:"chapter"`
	VerseStart *int `json:"verseStart"`
	VerseEnd   *int `json:"verseEnd"`
}

func matchChunkToVerse(ctx context.Context, client *gemini.Client, chunk string, bookList string) (*chunkMatch, error) {
	prompt := fmt.Sprintf(
		`Given this note text, identify which Bible book/chapter/verse range it discusses, if any.
Books (id:name): %s

Note text:
%s

Respond ONLY with JSON: {"bookId": <int or null>, "chapter": <int or null>, "verseStart": <int or null>, "verseEnd": <int or null>}
If the text isn't clearly about a specific passage, return all nulls.`,
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
	return &match, nil
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
