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

	var sb strings.Builder

	sb.WriteString("You are evaluating whether a user's summary correctly captures the key content of a Bible passage.\n")
	sb.WriteString("The user is NOT trying to recite the passage word-for-word — they are summarizing it in their own words.\n")
	sb.WriteString("Judge whether their summary demonstrates an accurate understanding of the passage's main events, ideas, or teachings.\n")
	sb.WriteString("Do not penalize different phrasing, paraphrasing, or concise wording.\n")
	sb.WriteString("Do penalize factual errors, misunderstandings, and failure to capture enough meaningful content to demonstrate understanding.\n\n")

	sb.WriteString("Scoring guidelines:\n")
	sb.WriteString("- First, determine a base score (1-10) strictly from the passage and the user's summary, as if no notes existed.\n")
	sb.WriteString("- Accuracy and meaningful passage-specific content are more important than completeness.\n")
	sb.WriteString("- A summary does NOT need to mention every event, detail, person, or teaching in the passage to receive a good score.\n")
	sb.WriteString("- Reward summaries that accurately capture multiple specific and meaningful elements of the passage, even when other important details are omitted.\n")
	sb.WriteString("- Missing some major events should reduce the score proportionally, but should not by itself make an otherwise accurate, passage-specific summary a very low score.\n")
	sb.WriteString("- A concise summary can score well if the content it includes is accurate and meaningfully represents the passage.\n")
	sb.WriteString("- Distinguish between an incomplete but passage-specific summary and a vague or generic summary.\n")
	sb.WriteString("- Reserve very low scores (1-3) for summaries that are substantially incorrect, extremely vague, contain very little passage-specific content, or demonstrate minimal understanding of the passage.\n")
	sb.WriteString("- A score of 4 or higher should generally indicate that the summary demonstrates real engagement with specific content from the passage, even if it is incomplete.\n")
	sb.WriteString("- Never fabricate that the summary covered content it did not actually mention.\n\n")
	sb.WriteString("- When a summary identifies a central theme and several specific events or details accurately, it should generally receive more than a middling score even if it omits other major events.\n")
	sb.WriteString("- Do not require a summary to cover most of the passage before giving it a score above 5.\n")
	sb.WriteString("- A summary's score should reflect both accuracy and the amount of meaningful understanding demonstrated, not simply the percentage of major events mentioned.\n")

	sb.WriteString("Notes handling:\n")
	sb.WriteString("- If no notes are provided for an item, treat the notes as completely absent.\n")
	sb.WriteString("- Never assume, infer, or hallucinate that notes exist.\n")
	sb.WriteString("- Notes may contain additional observations, interpretations, personal applications, or reflections about the passage.\n")
	sb.WriteString("- Notes may contribute to the score only through the bonus described below; they must never affect the base score.\n")
	sb.WriteString("- Notes must be relevant to the passage being evaluated to affect the score or be mentioned in feedback.\n")
	sb.WriteString("- If relevant notes support or reinforce the understanding demonstrated in the summary, you may mention that connection specifically.\n")
	sb.WriteString("- If relevant notes contain an important application, interpretation, or insight that is not reflected in the summary, you may specifically point out that connection and encourage the user to incorporate it into their summary.\n")
	sb.WriteString("- If relevant notes contain useful content that the summary does not capture, identify the specific idea rather than simply saying that notes are available.\n")
	sb.WriteString("- If notes are irrelevant to the passage, they must not affect the score or bonus and should not be mentioned in feedback.\n")
	sb.WriteString("- Never mention the bonus point, how many points the notes contributed, or that notes did or did not affect the score.\n\n")

	sb.WriteString("Feedback guidelines:\n")
	sb.WriteString("- In the \"feedback\" field, provide 1-2 constructive sentences about the user's summary.\n")
	sb.WriteString("- Clearly acknowledge what the user captured correctly before describing what could be improved.\n")
	sb.WriteString("- Base your description of what the user got right or wrong on their summary text. Never claim that the user included something that appears only in their notes.\n")
	sb.WriteString("- Do not treat a concise summary as incorrect simply because it does not include every event in the passage.\n")
	sb.WriteString("- When the summary accurately captures multiple specific events, people, ideas, or teachings, acknowledge that understanding even if other content is omitted.\n")
	sb.WriteString("- If relevant notes contain a personal application, reflection, or interpretation that directly connects to something in the summary, ALWAYS mention that connection in the feedback.\n")
	sb.WriteString("- When mentioning a personal application or reflection from the notes, state the specific insight rather than merely saying that notes are available.\n")
	sb.WriteString("- If the notes contain a meaningful insight that is not reflected in the summary, mention that insight as something the user could incorporate into their summary.\n")
	sb.WriteString("- Make clear that an insight mentioned from the notes comes from their notes and was not already included in their summary.\n")
	sb.WriteString("- If the notes are irrelevant to the passage, do not mention them.\n")
	sb.WriteString("- Do not mention notes merely because they exist; mention them when they contain a relevant personal application, reflection, interpretation, or insight that connects to the summary.\n")
	sb.WriteString("- Do not mention the scoring process, base score, bonus points, or how many points were gained or lost.\n\n")
	sb.WriteString("- When describing what the user captured, restate only what their summary actually says. Do not add reasons, causes, motivations, or details from the passage that the user did not write (for example, if they say a character wept, do not say what the character wept about).\n")
	sb.WriteString("- Put anything the user left out in the 'could improve' part of the feedback, not in the description of what they got right.\n")

	sb.WriteString(fmt.Sprintf(
		"Respond ONLY with a valid JSON array of exactly %d elements matching the order of items below:\n",
		count,
	))
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
