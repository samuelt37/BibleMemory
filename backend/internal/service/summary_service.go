package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/samuelt37/BibleMemory/internal/dto"
	"github.com/samuelt37/BibleMemory/internal/model"
	"github.com/samuelt37/BibleMemory/internal/repository"
)

type SummaryService struct {
	repo *repository.ScriptureRepository
}

func NewSummaryService(
	repo *repository.ScriptureRepository,
) *SummaryService {
	return &SummaryService{
		repo: repo,
	}
}

func (s *SummaryService) CheckSummary(req dto.SummaryRequest) ([]dto.SummaryResult, error) {
	if len(req.Answers) != len(req.Scripture.Ranges) {
		return nil, fmt.Errorf("answers count (%d) does not match ranges count (%d)", len(req.Answers), len(req.Scripture.Ranges))
	}
	if len(req.Scripture.Ranges) == 0 {
		return []dto.SummaryResult{}, nil
	}

	passages := make([]string, len(req.Scripture.Ranges))
	for i, rng := range req.Scripture.Ranges {
		singleRangeQuery := dto.ScriptureQuery{
			Translation: req.Scripture.Translation,
			Ranges:      []dto.ScriptureRange{rng},
		}

		verses, err := s.repo.GetScripture(singleRangeQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch range %d: %w", i, err)
		}
		passages[i] = concatVerses(verses)
	}

	return s.gradeAllWithAI(req.Answers, passages)
}

func concatVerses(verses []model.VerseInfo) string {
	var sb strings.Builder
	for i, v := range verses {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(v.Text)
	}
	return sb.String()
}

func (s *SummaryService) gradeAllWithAI(userAnswers, passages []string) ([]dto.SummaryResult, error) {
	count := len(userAnswers)

	var sb strings.Builder
	sb.WriteString("You are evaluating whether a user's summary correctly captures the key content of a Bible passage.\n")
	sb.WriteString("The user is NOT trying to recite the passage word-for-word — they are summarizing it in their own words.\n")
	sb.WriteString("Judge whether their summary reflects an accurate understanding of the passage's main events, ideas, or teachings.\n")
	sb.WriteString("Don't penalize different phrasing, paraphrasing, or omitted minor details — focus on whether the core meaning is correct.\n\n")
	sb.WriteString("Rate each summary's accuracy from 1 to 10, where 10 means it fully and correctly captures the passage's key content,\n")
	sb.WriteString("and 1 means it's missing or misrepresents the content entirely.\n\n")
	sb.WriteString(fmt.Sprintf("Respond ONLY with a valid JSON array of exactly %d elements matching the order of items below:\n", count))
	sb.WriteString(`[{"accuracy": 8, "feedback": "..."}, {"accuracy": 9, "feedback": "..."}]` + "\n\n")
	sb.WriteString("Items to evaluate:\n")

	for i := 0; i < count; i++ {
		sb.WriteString(fmt.Sprintf("--- Item %d ---\nPassage text: %s\nUser's summary: %s\n\n", i+1, passages[i], userAnswers[i]))
	}

	body := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]string{
					{"text": sb.String()},
				},
			},
		},
		"generationConfig": map[string]any{
			"response_mime_type": "application/json",
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("API_KEY environment variable is not set on the server")
	}

	primaryModel := os.Getenv("GEMINI_MODEL")
	if primaryModel == "" {
		primaryModel = "gemini-3.6-flash"
	}

	modelsToTry := []string{primaryModel}
	for _, fallback := range []string{"gemini-3.5-flash", "gemini-3.7-flash"} {
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

			url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", modelName)
			httpReq, err := http.NewRequest("POST", url, bytes.NewReader(payload))
			if err != nil {
				return nil, err
			}

			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("x-goog-api-key", apiKey)

			resp, err := http.DefaultClient.Do(httpReq)
			if err != nil {
				lastErr = err
				continue
			}

			if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusTooManyRequests {
				var errBody bytes.Buffer
				errBody.ReadFrom(resp.Body)
				resp.Body.Close()
				lastErr = fmt.Errorf("Gemini API returned status %d for model %s: %s", resp.StatusCode, modelName, errBody.String())
				continue
			}

			if resp.StatusCode == http.StatusNotFound {
				var errBody bytes.Buffer
				errBody.ReadFrom(resp.Body)
				resp.Body.Close()
				lastErr = fmt.Errorf("Gemini API returned status 404 for model %s: %s", modelName, errBody.String())
				break
			}

			if resp.StatusCode != http.StatusOK {
				var errBody bytes.Buffer
				errBody.ReadFrom(resp.Body)
				resp.Body.Close()
				return nil, fmt.Errorf("Gemini API returned status %d: %s", resp.StatusCode, errBody.String())
			}

			var apiResp struct {
				Candidates []struct {
					Content struct {
						Parts []struct {
							Text string `json:"text"`
						} `json:"parts"`
					} `json:"content"`
				} `json:"candidates"`
			}
			err = json.NewDecoder(resp.Body).Decode(&apiResp)
			resp.Body.Close()
			if err != nil {
				return nil, err
			}
			if len(apiResp.Candidates) == 0 || len(apiResp.Candidates[0].Content.Parts) == 0 {
				return nil, fmt.Errorf("empty response from Gemini")
			}

			rawText := strings.TrimSpace(apiResp.Candidates[0].Content.Parts[0].Text)
			var results []dto.SummaryResult
			if err := json.Unmarshal([]byte(rawText), &results); err != nil {
				var single dto.SummaryResult
				if err2 := json.Unmarshal([]byte(rawText), &single); err2 == nil {
					results = []dto.SummaryResult{single}
				} else {
					return nil, fmt.Errorf("failed to parse Gemini response: %w (raw: %s)", err, rawText)
				}
			}

			for len(results) < count {
				results = append(results, dto.SummaryResult{
					Accuracy: 5,
					Feedback: "Passage evaluated.",
				})
			}
			if len(results) > count {
				results = results[:count]
			}

			return results, nil
		}
	}

	return nil, fmt.Errorf("grading failed after retries: %w", lastErr)
}
