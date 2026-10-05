package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 45 * time.Second, // Safe timeout for large PDFs and LLM responses
		},
	}
}

func (c *Client) getAPIKey() (string, error) {
	if c.apiKey != "" {
		return c.apiKey, nil
	}
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return "", fmt.Errorf("API_KEY environment variable is not set on the server")
	}
	return apiKey, nil
}

func (c *Client) modelsToTry() []string {
	primaryModel := os.Getenv("GEMINI_MODEL")
	if primaryModel == "" {
		primaryModel = "gemini-3.6-flash"
	}

	models := []string{primaryModel}
	for _, fallback := range []string{"gemini-flash-lite-latest", "gemini-3.5-flash-lite"} {
		if fallback != primaryModel {
			models = append(models, fallback)
		}
	}
	return models
}

// GenerateContent sends a payload to Gemini with fallback models and retry handling.
func (c *Client) GenerateContent(ctx context.Context, body any) (string, error) {
	apiKey, err := c.getAPIKey()
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request payload: %w", err)
	}

	models := c.modelsToTry()
	var lastErr error

	for _, modelName := range models {
		for attempt := 0; attempt < 2; attempt++ {
			if attempt > 0 {
				select {
				case <-ctx.Done():
					return "", ctx.Err()
				case <-time.After(300 * time.Millisecond):
				}
			}

			url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", modelName)
			req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
			if err != nil {
				return "", err
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("x-goog-api-key", apiKey)

			resp, err := c.httpClient.Do(req)
			if err != nil {
				lastErr = err
				continue
			}

			// 503 (overloaded) or 429 (rate limited) or 404 (not found): failover to next model immediately
			if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusNotFound {
				bodyBytes, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				lastErr = fmt.Errorf("gemini API returned status %d for model %s: %s", resp.StatusCode, modelName, string(bodyBytes))
				break
			}

			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				return "", fmt.Errorf("gemini API returned status %d for model %s: %s", resp.StatusCode, modelName, string(bodyBytes))
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

			err = json.NewDecoder(resp.Body).Decode(&result)
			resp.Body.Close()
			if err != nil {
				return "", fmt.Errorf("failed to decode gemini response: %w", err)
			}

			if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
				return "", fmt.Errorf("empty candidates in gemini response")
			}

			return result.Candidates[0].Content.Parts[0].Text, nil
		}
	}

	return "", fmt.Errorf("gemini generation failed after retries: %w", lastErr)
}

// EmbedText generates a vector embedding for text using gemini-embedding-001.
func (c *Client) EmbedText(ctx context.Context, text string, dimensions int) ([]float32, error) {
	apiKey, err := c.getAPIKey()
	if err != nil {
		return nil, err
	}

	body := map[string]any{
		"model": "models/gemini-embedding-001",
		"content": map[string]any{
			"parts": []map[string]string{{"text": text}},
		},
		"outputDimensionality": dimensions,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-embedding-001:embedContent"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", apiKey)

	resp, err := c.httpClient.Do(req)
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
