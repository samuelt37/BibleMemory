package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

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

	results := make([]dto.SummaryResult, len(req.Scripture.Ranges))
	for i, rng := range req.Scripture.Ranges {
		singleRangeQuery := dto.ScriptureQuery{
			Translation: req.Scripture.Translation,
			Ranges:      []dto.ScriptureRange{rng},
		}

		verses, err := s.repo.GetScripture(singleRangeQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch range %d: %w", i, err)
		}

		realText := concatVerses(verses)

		result, err := s.gradeWithAI(req.Answers[i], realText)
		if err != nil {
			return nil, fmt.Errorf("failed to grade range %d: %w", i, err)
		}
		results[i] = result
	}

	return results, nil
}

func concatVerses(verses []model.Verse) string {
	var sb strings.Builder
	for i, v := range verses {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(v.Text)
	}
	return sb.String()
}

func (s *SummaryService) gradeWithAI(userAnswer, realText string) (dto.SummaryResult, error) {
	prompt := fmt.Sprintf(
		"You are evaluating whether a user's summary correctly captures the key content of a Bible passage. "+
			"The user is NOT trying to recite the passage word-for-word — they are summarizing it in their own words. "+
			"Judge whether their summary reflects an accurate understanding of the passage's main events, ideas, or teachings. "+
			"Don't penalize different phrasing, paraphrasing, or omitted minor details — focus on whether the core meaning is correct.\n\n"+
			"Rate the summary's accuracy from 1 to 10, where 10 means it fully and correctly captures the passage's key content, "+
			"and 1 means it's missing or misrepresents the content entirely.\n\n"+
			"Respond ONLY with valid JSON, no markdown, no explanation outside the JSON, in exactly this shape:\n"+
			`{"accuracy": 8, "feedback": "..."}`+"\n\n"+
			"Passage text: %s\nUser's summary: %s\n",
		realText, userAnswer,
	)

	body := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]any{
			"response_mime_type": "application/json",
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return dto.SummaryResult{}, err
	}

	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.6-flash:generateContent"
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return dto.SummaryResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", os.Getenv("API_KEY"))

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return dto.SummaryResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return dto.SummaryResult{}, fmt.Errorf("Gemini API returned status %d", resp.StatusCode)
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
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return dto.SummaryResult{}, err
	}
	if len(apiResp.Candidates) == 0 || len(apiResp.Candidates[0].Content.Parts) == 0 {
		return dto.SummaryResult{}, fmt.Errorf("empty response from Gemini")
	}

	var result dto.SummaryResult
	if err := json.Unmarshal([]byte(apiResp.Candidates[0].Content.Parts[0].Text), &result); err != nil {
		return dto.SummaryResult{}, fmt.Errorf("failed to parse Gemini response: %w", err)
	}

	return result, nil
}
