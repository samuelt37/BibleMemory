package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/samuelt37/BibleMemory/internal/dto"
	"github.com/samuelt37/BibleMemory/internal/model"
	"github.com/samuelt37/BibleMemory/internal/repository"
)

type SummaryService struct {
	repo      *repository.ScriptureRepository
	chunkRepo *repository.NoteChunkRepository
}

func NewSummaryService(
	repo *repository.ScriptureRepository,
	chunkRepo *repository.NoteChunkRepository,
) *SummaryService {
	return &SummaryService{
		repo:      repo,
		chunkRepo: chunkRepo,
	}
}

func (s *SummaryService) CheckSummary(req dto.SummaryRequest, userID int) ([]dto.SummaryResult, error) {
	if len(req.Answers) != len(req.Scripture.Ranges) {
		return nil, fmt.Errorf("answers count (%d) does not match ranges count (%d)", len(req.Answers), len(req.Scripture.Ranges))
	}
	if len(req.Scripture.Ranges) == 0 {
		return []dto.SummaryResult{}, nil
	}

	passages := make([]string, len(req.Scripture.Ranges))
	notesContext := make([]string, len(req.Scripture.Ranges))

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

		if userID != 0 {
			endBook := rng.Start.Book
			if rng.End != nil {
				endBook = rng.End.Book
			}
			chunks, err := s.chunkRepo.FindRelevant(userID, rng.Start.Book, endBook)
			if err != nil {
				log.Printf("CheckSummary: error finding notes for user %d (books %d-%d): %v", userID, rng.Start.Book, endBook, err)
			} else {
				log.Printf("CheckSummary: found %d note chunk(s) for user %d (books %d-%d)", len(chunks), userID, rng.Start.Book, endBook)
				if len(chunks) > 0 {
					notesContext[i] = strings.Join(chunks, "\n---\n")
				}
			}
		} else {
			log.Println("CheckSummary: userID is 0 (unauthenticated), skipping notes retrieval")
		}
	}

	return s.gradeAllWithAI(req.Answers, passages, notesContext)
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

func (s *SummaryService) gradeAllWithAI(userAnswers, passages, notesContext []string) ([]dto.SummaryResult, error) {
	count := len(userAnswers)

	var sb strings.Builder
	sb.WriteString("You are evaluating whether a user's summary correctly captures the key content of a Bible passage.\n")
	sb.WriteString("The user is NOT trying to recite the passage word-for-word — they are summarizing it in their own words.\n")
	sb.WriteString("Judge whether their summary reflects an accurate understanding of the passage's main events, ideas, or teachings.\n")
	sb.WriteString("Don't penalize different phrasing, paraphrasing, or omitted minor details — focus on whether the core meaning is correct.\n\n")

	sb.WriteString("Scoring guidelines:\n")
	sb.WriteString("- Rate accuracy from 1 to 10 based strictly on how well the summary reflects the actual passage text.\n\n")

	sb.WriteString("Feedback guidelines:\n")
	sb.WriteString("- In the \"feedback\" field, provide 1-2 constructive sentences on their summary.\n")
	sb.WriteString("- If \"User's own notes on this passage\" are present, YOU MUST explicitly reference them in the feedback (e.g., \"As you noted in your personal notes...\" or \"This ties directly into your application regarding...\").\n")
	sb.WriteString("- If no notes are provided, base the feedback solely on the passage and summary.\n\n")

	sb.WriteString(fmt.Sprintf("Respond ONLY with a valid JSON array of exactly %d elements matching the order of items below:\n", count))
	sb.WriteString(`[{"accuracy": 8, "feedback": "..."}, {"accuracy": 9, "feedback": "..."}]` + "\n\n")
	sb.WriteString("Items to evaluate:\n")

	for i := 0; i < count; i++ {
		sb.WriteString(fmt.Sprintf("--- Item %d ---\nPassage text: %s\n", i+1, passages[i]))
		if notesContext[i] != "" {
			sb.WriteString(fmt.Sprintf("User's own notes on this passage: %s\n", notesContext[i]))
		}
		sb.WriteString(fmt.Sprintf("User's summary: %s\n\n", userAnswers[i]))
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

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

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
		for attempt := 0; attempt < 2; attempt++ {
			if attempt > 0 {
				time.Sleep(500 * time.Millisecond)
			}

			url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", modelName)
			httpReq, err := http.NewRequest("POST", url, bytes.NewReader(payload))
			if err != nil {
				return nil, err
			}

			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("x-goog-api-key", apiKey)

			resp, err := httpClient.Do(httpReq)
			if err != nil {
				lastErr = err
				continue
			}

			if resp.StatusCode == http.StatusServiceUnavailable {
				var errBody bytes.Buffer
				errBody.ReadFrom(resp.Body)
				resp.Body.Close()
				lastErr = fmt.Errorf("Gemini API returned status 503 (high demand) for model %s: %s", modelName, errBody.String())
				// Overloaded model won't recover in 1s; failover to next model immediately
				break
			}

			if resp.StatusCode == http.StatusTooManyRequests {
				var errBody bytes.Buffer
				errBody.ReadFrom(resp.Body)
				resp.Body.Close()
				lastErr = fmt.Errorf("Gemini API returned status 429 (rate limit) for model %s: %s", modelName, errBody.String())
				break
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
