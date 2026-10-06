package summary

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/samuelt37/BibleMemory/internal/gemini"
	"github.com/samuelt37/BibleMemory/internal/notes"
	"github.com/samuelt37/BibleMemory/internal/scripture"
)

type Service struct {
	repo      *scripture.Repository
	chunkRepo *notes.ChunkRepository
	gemini    *gemini.Client
}

func NewService(
	repo *scripture.Repository,
	chunkRepo *notes.ChunkRepository,
) *Service {
	return &Service{
		repo:      repo,
		chunkRepo: chunkRepo,
		gemini:    gemini.NewClient(),
	}
}

func (s *Service) CheckSummary(ctx context.Context, req Request, userID int) ([]Result, error) {
	if len(req.Answers) != len(req.Scripture.Ranges) {
		return nil, fmt.Errorf("answers count (%d) does not match ranges count (%d)", len(req.Answers), len(req.Scripture.Ranges))
	}
	if len(req.Scripture.Ranges) == 0 {
		return []Result{}, nil
	}

	passages := make([]string, len(req.Scripture.Ranges))
	notesContext := make([]string, len(req.Scripture.Ranges))
	noteRefs := make([][]NoteRef, len(req.Scripture.Ranges))

	for i, rng := range req.Scripture.Ranges {
		singleRangeQuery := scripture.Query{
			Translation: req.Scripture.Translation,
			Ranges:      []scripture.Range{rng},
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

			// Fast path: fetch chunks by recency (<2ms DB query)
			chunks, err := s.chunkRepo.FindRelevant(userID, rng.Start.Book, endBook, nil)
			if err != nil {
				log.Printf("CheckSummary: error finding notes for user %d (books %d-%d): %v", userID, rng.Start.Book, endBook, err)
			} else {
				// Only perform vector similarity embedding if the user has more than 5 notes
				if len(chunks) == 5 {
					if total, _ := s.chunkRepo.CountByBooks(userID, rng.Start.Book, endBook); total > 5 {
						emb, err := s.gemini.EmbedText(ctx, passages[i], 768)
						if err == nil && emb != nil {
							if vecChunks, err := s.chunkRepo.FindRelevant(userID, rng.Start.Book, endBook, emb); err == nil && len(vecChunks) > 0 {
								chunks = vecChunks
							}
						}
					}
				}
				if len(chunks) > 0 {
					parts := make([]string, 0, len(chunks))
					seen := make(map[int]bool)
					for _, c := range chunks {
						parts = append(parts, fmt.Sprintf("[Note %d: %s]\n%s", c.NoteID, c.Title, c.Content))
						if !seen[c.NoteID] {
							seen[c.NoteID] = true
							noteRefs[i] = append(noteRefs[i], NoteRef{NoteID: c.NoteID, Title: c.Title})
						}
					}
					notesContext[i] = strings.Join(parts, "\n---\n")
				}
			}
		} else {
			log.Println("CheckSummary: userID is 0 (unauthenticated), skipping notes retrieval")
		}
	}

	results, err := s.gradeAllWithAI(req.Answers, passages, notesContext)
	if err != nil {
		return nil, err
	}
	for i := range results {
		if i < len(noteRefs) {
			uses := make(map[int]NoteUse, len(results[i].NotesUsed))
			for _, u := range results[i].NotesUsed {
				uses[u.ID] = u
			}
			for _, ref := range noteRefs[i] {
				u, ok := uses[ref.NoteID]
				if !ok {
					continue
				}
				ref.Quote = matchQuote(results[i].Feedback, u.Quote)
				ref.SummaryQuote = matchQuote(req.Answers[i], u.SummaryQuote)
				results[i].Notes = append(results[i].Notes, ref)
			}
		}
		results[i].NotesUsed = nil
	}
	return results, nil
}

func concatVerses(verses []scripture.VerseInfo) string {
	var sb strings.Builder
	for i, v := range verses {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(v.Text)
	}
	return sb.String()
}

func (s *Service) gradeAllWithAI(userAnswers, passages, notesContext []string) ([]Result, error) {
	count := len(userAnswers)
	prompt := buildGradingPrompt(userAnswers, passages, notesContext)
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
			"temperature":        0.2,
		},
	}
	rawText, err := s.gemini.GenerateContent(context.Background(), body)
	if err != nil {
		return nil, err
	}

	rawText = strings.TrimSpace(rawText)
	var results []Result
	if err := json.Unmarshal([]byte(rawText), &results); err != nil {
		var single Result
		if err2 := json.Unmarshal([]byte(rawText), &single); err2 == nil {
			results = []Result{single}
		} else {
			return nil, fmt.Errorf("failed to parse Gemini response: %w (raw: %s)", err, rawText)
		}
	}

	for len(results) < count {
		results = append(results, Result{
			Accuracy: 5,
			Feedback: "Passage evaluated.",
		})
	}
	if len(results) > count {
		results = results[:count]
	}

	return results, nil
}

func matchQuote(feedback, quote string) string {
	q := strings.TrimSpace(quote)
	if q == "" {
		return ""
	}
	if strings.Contains(feedback, q) {
		return q
	}
	q = strings.TrimRight(q, ".,;:!? ")
	if q != "" && strings.Contains(feedback, q) {
		return q
	}
	return ""
}
