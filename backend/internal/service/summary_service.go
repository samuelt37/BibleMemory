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
	sb.WriteString("Don't penalize different phrasing or paraphrasing — but DO penalize missing major events, people, or teachings.\n\n")

	sb.WriteString("Notes handling:\n")
	sb.WriteString("- If no notes are provided for an item, treat the notes as completely absent.\n")
	sb.WriteString("- Never assume, infer, or hallucinate that notes exist.\n")
	sb.WriteString("- If notes are absent, the score MUST be based only on the passage and summary.\n")
	sb.WriteString("- If notes are absent, the feedback MUST NOT mention notes, note availability, or notes affecting the score.\n")
	sb.WriteString("- Only discuss notes when notes are explicitly included in that item's input.\n\n")

	sb.WriteString("Scoring guidelines:\n")
	sb.WriteString("- First, determine a base score (1-10) strictly on how well the summary reflects the actual passage text,\n")
	sb.WriteString("  judged as if no notes existed.\n")
	sb.WriteString("- If the base score is 4 or higher (the summary shows real engagement with multiple specific elements of\n")
	sb.WriteString("  the passage, not just a single generic clause) AND the user has notes that align with their summary,\n")
	sb.WriteString("  you may add 1 point as a bonus, capped at 10 total.\n")
	sb.WriteString("- If the base score is 3 or lower (the summary is a single generic statement that could describe almost\n")
	sb.WriteString("  any passage, e.g. \"God created the universe\" for a chapter with dozens of specific details), apply\n")
	sb.WriteString("  ZERO bonus regardless of notes. A summary this thin has not demonstrated engagement worth rewarding,\n")
	sb.WriteString("  even if the user's notes are detailed.\n")
	sb.WriteString("- Never fabricate that the summary covered content it didn't.\n\n")

	sb.WriteString("Feedback guidelines:\n")
	sb.WriteString("- In the \"feedback\" field, provide 1-2 constructive sentences on their summary.\n")
	sb.WriteString("- Base your description of what they got right or wrong on their summary text — never describe passage\n")
	sb.WriteString("  details that only appear in their notes as if the user wrote them in their summary.\n")
	sb.WriteString("- If notes are explicitly provided AND a bonus point was actually applied (base score was 4+), you may mention\n")
	sb.WriteString("  that notes are available for this passage and factored into the score.\n")
	sb.WriteString("- If notes are explicitly provided but NO bonus was applied (base score was 3 or lower), you may mention\n")
	sb.WriteString("  that notes are available, but do NOT say they factored into the score.\n")
	sb.WriteString("- If NO notes are provided, do not mention notes anywhere in the feedback. Do not say notes are available,\n")
	sb.WriteString("  missing, unavailable, or factored into the score.\n")
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
