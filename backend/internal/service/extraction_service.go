package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func extractTextFromFile(ctx context.Context, fileBytes []byte, mimeType string) (string, error) {
	apiKey, err := getGeminiAPIKey()
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(fileBytes)

	body, _ := json.Marshal(map[string]any{
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
	})

	primaryModel := os.Getenv("GEMINI_MODEL")
	if primaryModel == "" {
		primaryModel = "gemini-3.6-flash"
	}

	modelsToTry := []string{primaryModel}
	for _, fallback := range []string{"gemini-flash-lite-latest", "gemini-3.5-flash-lite"} {
		if fallback != primaryModel {
			modelsToTry = append(modelsToTry, fallback)
		}
	}

	var lastErr error
	for _, modelName := range modelsToTry {
		for attempt := 0; attempt < 3; attempt++ {
			if attempt > 0 {
				time.Sleep(time.Duration(attempt*1500) * time.Millisecond)
			}

			url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, apiKey)
			req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
			if err != nil {
				return "", err
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				lastErr = err
				continue
			}

			if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusTooManyRequests {
				bodyBytes, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				lastErr = fmt.Errorf("gemini API returned status %d for model %s: %s", resp.StatusCode, modelName, string(bodyBytes))
				continue
			}

			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				return "", fmt.Errorf("gemini extraction API returned status %d: %s", resp.StatusCode, string(bodyBytes))
			}

			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

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
				return "", fmt.Errorf("failed to decode extraction response: %w, body: %s", err, string(bodyBytes))
			}
			if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
				return "", fmt.Errorf("no candidates in extraction response: %s", string(bodyBytes))
			}

			return result.Candidates[0].Content.Parts[0].Text, nil
		}
	}

	return "", fmt.Errorf("extraction failed after retries: %w", lastErr)
}
