package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func getGeminiAPIKey() (string, error) {
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return "", fmt.Errorf("API_KEY environment variable is not set on the server")
	}
	return apiKey, nil
}

type chunkMatch struct {
	BookID     *int `json:"bookId"`
	Chapter    *int `json:"chapter"`
	VerseStart *int `json:"verseStart"`
	VerseEnd   *int `json:"verseEnd"`
}

func matchChunkToVerse(ctx context.Context, chunk string, bookList string) (*chunkMatch, error) {
	apiKey, err := getGeminiAPIKey()
	if err != nil {
		return nil, err
	}

	prompt := fmt.Sprintf(
		`Given this note text, identify which Bible book/chapter/verse range it discusses, if any.
Books (id:name): %s

Note text:
%s

Respond ONLY with JSON: {"bookId": <int or null>, "chapter": <int or null>, "verseStart": <int or null>, "verseEnd": <int or null>}
If the text isn't clearly about a specific passage, return all nulls.`,
		bookList, chunk,
	)

	body, _ := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": prompt}}},
		},
	})

	models := []string{"gemini-3.6-flash", "gemini-flash-lite-latest"}
	var bodyBytes []byte
	var lastStatus int

	for _, model := range models {
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		bodyBytes, _ = io.ReadAll(resp.Body)
		lastStatus = resp.StatusCode
		if resp.StatusCode == http.StatusOK {
			break
		}
	}

	if lastStatus != http.StatusOK {
		return nil, fmt.Errorf("gemini API returned status %d: %s", lastStatus, string(bodyBytes))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to decode gemini response: %w, body: %s", err, string(bodyBytes))
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no candidates in gemini response: %s", string(bodyBytes))
	}

	raw := cleanJSONFence(result.Candidates[0].Content.Parts[0].Text)

	var match chunkMatch
	if err := json.Unmarshal([]byte(raw), &match); err != nil {
		return nil, fmt.Errorf("failed to parse match response: %w", err)
	}
	return &match, nil
}

func embedChunk(ctx context.Context, text string) ([]float32, error) {
	apiKey, err := getGeminiAPIKey()
	if err != nil {
		return nil, err
	}

	body, _ := json.Marshal(map[string]any{
		"model": "models/gemini-embedding-001",
		"content": map[string]any{
			"parts": []map[string]string{{"text": text}},
		},
		"outputDimensionality": 768, // matches VECTOR(768) in your schema
	})

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-embedding-001:embedContent?key=%s", apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini embedding API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Embedding struct {
			Values []float32 `json:"values"`
		} `json:"embedding"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w, body: %s", err, string(bodyBytes))
	}
	return result.Embedding.Values, nil
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
