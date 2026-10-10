package notes

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/google/uuid"
	"github.com/samuelt37/BibleMemory/internal/gemini"
	"github.com/samuelt37/BibleMemory/internal/scripture"
	"github.com/samuelt37/BibleMemory/internal/storage"
)

var (
	ErrNotFound = errors.New("not found")
)

type Service struct {
	repo          *Repository
	chunkRepo     *ChunkRepository
	scriptureRepo *scripture.Repository
	r2            *storage.R2Client
	gemini        *gemini.Client

	bookListCache   string
	bookListCacheMu sync.Mutex
	matchLimiter    *rate.Limiter
}

func NewService(repo *Repository, chunkRepo *ChunkRepository, scriptureRepo *scripture.Repository, r2 *storage.R2Client) *Service {
	return &Service{
		repo:          repo,
		chunkRepo:     chunkRepo,
		scriptureRepo: scriptureRepo,
		r2:            r2,
		gemini:        gemini.NewClient(),
		matchLimiter:  rate.NewLimiter(rate.Every(5*time.Second), 2),
	}
}

func (s *Service) buildBookList() (string, error) {
	s.bookListCacheMu.Lock()
	defer s.bookListCacheMu.Unlock()

	if s.bookListCache != "" {
		return s.bookListCache, nil
	}

	books, err := s.scriptureRepo.GetBooks()
	if err != nil {
		return "", err
	}

	var parts []string
	for _, b := range books {
		parts = append(parts, fmt.Sprintf("%d:%s", b.ID, b.Book))
	}

	s.bookListCache = strings.Join(parts, ",")
	return s.bookListCache, nil
}

func (s *Service) UploadNote(ctx context.Context, userID int, file multipart.File, header *multipart.FileHeader) (*Note, error) {
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("users/%d/%s-%s", userID, uuid.NewString(), header.Filename)
	mimeType := header.Header.Get("Content-Type")

	if err := s.r2.Upload(ctx, key, bytes.NewReader(fileBytes), mimeType); err != nil {
		return nil, err
	}

	note, err := s.repo.CreateNote(userID, header.Filename, key, mimeType, header.Size)
	if err != nil {
		return nil, err
	}

	go s.extractAndProcess(context.Background(), userID, note.ID, fileBytes, mimeType)

	return note, nil
}

func (s *Service) extractAndProcess(ctx context.Context, userID, noteID int, fileBytes []byte, mimeType string) {
	log.Println("extractAndProcess: starting for note", noteID, "size:", len(fileBytes), "bytes")
	text, err := extractTextFromFile(ctx, s.gemini, fileBytes, mimeType)
	if err != nil {
		log.Println("extractAndProcess: extraction failed for note", noteID, ":", err)
		s.repo.UpdateStatus(noteID, "failed")
		return
	}

	if err := s.repo.UpdateRawText(noteID, text); err != nil {
		log.Println("extractAndProcess: failed to save extracted text for note", noteID, ":", err)
		s.repo.UpdateStatus(noteID, "failed")
		return
	}

	if err := s.ProcessNote(ctx, userID, noteID); err != nil {
		log.Println("extractAndProcess: ProcessNote failed for note", noteID, ":", err)
	}
}

func (s *Service) CreateTextNote(userID int, title, text string) (*Note, error) {
	filename := title
	if filename == "" {
		filename = "Untitled note"
	}
	note, err := s.repo.CreateTextNote(userID, filename, text)
	if err != nil {
		return nil, err
	}

	go s.ProcessNote(context.Background(), userID, note.ID)

	return note, nil
}

func (s *Service) ListNotes(ctx context.Context, userID int, f NoteFilter) ([]Note, error) {
	keyword, err := s.repo.ListByUser(userID, f)
	if err != nil {
		return nil, err
	}

	// no search text: keyword/passage filtering is the whole answer
	if len(strings.TrimSpace(f.Query)) < 3 {
		return keyword, nil
	}

	// meaning-based matches; any failure falls back to keyword results
	emb, err := EmbedText(ctx, f.Query)
	if err != nil {
		log.Printf("ListNotes: embed failed, keyword only: %v", err)
		return keyword, nil
	}
	ids, err := s.chunkRepo.SearchNoteIDs(userID, emb, f.BookID, f.Chapter, searchMaxDist, 20)
	if err != nil {
		log.Printf("ListNotes: vector search failed, keyword only: %v", err)
		return keyword, nil
	}

	seen := make(map[int]bool, len(keyword))
	for _, n := range keyword {
		seen[n.ID] = true
	}
	var extraIDs []int
	for _, id := range ids {
		if !seen[id] {
			extraIDs = append(extraIDs, id)
		}
	}
	extra, err := s.repo.ListByIDs(userID, extraIDs)
	if err != nil {
		log.Printf("ListNotes: load similar notes failed: %v", err)
		return keyword, nil
	}
	byID := make(map[int]Note, len(extra))
	for _, n := range extra {
		byID[n.ID] = n
	}

	merged := keyword
	for _, id := range extraIDs { // ranked order from the vector search
		if n, ok := byID[id]; ok {
			merged = append(merged, n)
		}
	}
	return merged, nil
}

const searchMaxDist = 0.45

func (s *Service) GetDownloadURL(ctx context.Context, userID, noteID int) (string, error) {
	note, err := s.repo.GetByID(userID, noteID)
	if err != nil {
		return "", err
	}
	if note == nil {
		return "", fmt.Errorf("note not found")
	}
	return s.r2.PresignedGetURL(ctx, *note.R2Key, 15*time.Minute)
}

func (s *Service) DeleteNote(ctx context.Context, userID, noteID int) error {
	note, err := s.repo.GetByID(userID, noteID)
	if err != nil {
		return err
	}
	if note == nil {
		return fmt.Errorf("note not found")
	}

	if note.SourceType == "file" && note.R2Key != nil {
		if err := s.r2.Delete(ctx, *note.R2Key); err != nil {
			return err
		}
	}

	return s.repo.Delete(userID, noteID)
}

func (s *Service) ProcessNote(ctx context.Context, userID, noteID int) error {
	note, err := s.repo.GetByID(userID, noteID)
	if err != nil || note == nil {
		return fmt.Errorf("note not found")
	}
	if note.RawText == nil {
		return fmt.Errorf("note has no text to process yet")
	}

	chunks := chunkText(*note.RawText, 400)
	log.Println("ProcessNote: chunked into", len(chunks), "pieces")

	bookList, err := s.buildBookList()
	if err != nil {
		s.repo.UpdateStatus(noteID, "failed")
		return err
	}

	results := make([]chunkResult, len(chunks))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)

	for i, chunk := range chunks {
		wg.Add(1)
		go func(idx int, c string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			match, err := s.matchChunk(ctx, c, bookList)
			if err != nil {
				results[idx] = chunkResult{content: c, err: err}
				return
			}
			embedding, err := s.gemini.EmbedText(ctx, c, 768)
			if err != nil {
				results[idx] = chunkResult{content: c, err: err}
				return
			}
			results[idx] = chunkResult{content: c, match: match, embedding: embedding}
		}(i, chunk)
	}
	wg.Wait()

	for i, r := range results {
		if r.err != nil {
			log.Println("ProcessNote: chunk", i, "failed:", r.err)
			s.repo.UpdateStatus(noteID, "failed")
			return r.err
		}
		if err := s.chunkRepo.Create(userID, noteID, r.content, r.embedding, validRefs(r.match.Refs)); err != nil {
			log.Println("ProcessNote: insert failed for chunk", i, ":", err)
			s.repo.UpdateStatus(noteID, "failed")
			return err
		}
	}

	log.Println("ProcessNote: done, status -> ready")
	return s.repo.UpdateStatus(noteID, "ready")
}

func (s *Service) GetNote(userID, noteID int) (*Note, error) {
	return s.repo.GetByID(userID, noteID)
}

func (s *Service) EmbedText(ctx context.Context, text string) ([]float32, error) {
	return s.gemini.EmbedText(ctx, text, 768)
}

func validRefs(refs []verseRef) []verseRef {
	out := make([]verseRef, 0, len(refs))
	for _, r := range refs {
		if r.BookID == nil {
			continue
		}
		if r.Chapter == nil && (r.VerseStart != nil || r.VerseEnd != nil) {
			continue
		}
		if r.VerseStart != nil && r.VerseEnd != nil && *r.VerseEnd < *r.VerseStart {
			continue
		}
		out = append(out, r)
	}
	return out
}

func (s *Service) ReRefNotes(ctx context.Context, userID, noteID int) error {
	note, err := s.repo.GetByID(userID, noteID)
	if err != nil {
		return err
	}
	if note == nil {
		return err
	}

	chunks, err := s.chunkRepo.ListByNote(noteID)
	if err != nil {
		return err
	}
	if len(chunks) == 0 {
		return fmt.Errorf("note doesn't exist or isn't theirs")
	}

	bookList, err := s.buildBookList()
	if err != nil {
		s.repo.UpdateStatus(noteID, "failed")
		return err
	}

	for _, c := range chunks {
		match, err := s.matchChunk(ctx, c.Content, bookList)
		if err != nil || match == nil {
			return fmt.Errorf("chunk %d: %w", c.ID, err) // leave remaining chunks untouched
		}

		if err := s.chunkRepo.ReplaceRefs(userID, noteID, c.ID, validRefs(match.Refs)); err != nil {
			return err
		}
	}
	return nil
}

var ErrRateLimited = errors.New("gemini rate limited")

func (s *Service) matchChunk(ctx context.Context, content, bookList string) (*chunkMatch, error) {
	for attempt := 0; attempt < 2; attempt++ {
		if err := s.matchLimiter.Wait(ctx); err != nil {
			return nil, err
		}
		match, err := matchChunkToVerse(ctx, s.gemini, content, bookList)
		if err == nil {
			return match, nil
		}

		if !isRateLimited(err) {
			return nil, err
		}
		if attempt == 0 {
			// Google said to retry in ~50s; wait it out once
			select {
			case <-time.After(55 * time.Second):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
	return nil, ErrRateLimited
}

func isRateLimited(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "RESOURCE_EXHAUSTED") || strings.Contains(msg, "429")
}
