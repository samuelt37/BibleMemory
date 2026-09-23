package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/samuelt37/BibleMemory/internal/model"
	"github.com/samuelt37/BibleMemory/internal/repository"
	"github.com/samuelt37/BibleMemory/internal/storage"
)

type NoteService struct {
	repo          *repository.NoteRepository
	chunkRepo     *repository.NoteChunkRepository
	scriptureRepo *repository.ScriptureRepository
	r2            *storage.R2Client

	bookListCache   string
	bookListCacheMu sync.Mutex
}

func NewNoteService(repo *repository.NoteRepository, chunkRepo *repository.NoteChunkRepository, scriptureRepo *repository.ScriptureRepository, r2 *storage.R2Client) *NoteService {
	return &NoteService{repo: repo, chunkRepo: chunkRepo, scriptureRepo: scriptureRepo, r2: r2}
}

func (s *NoteService) buildBookList() (string, error) {
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

func (s *NoteService) UploadNote(ctx context.Context, userID int, file multipart.File, header *multipart.FileHeader) (*model.Note, error) {
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

func (s *NoteService) extractAndProcess(ctx context.Context, userID, noteID int, fileBytes []byte, mimeType string) {
	log.Println("extractAndProcess: starting for note", noteID, "size:", len(fileBytes), "bytes")
	text, err := extractTextFromFile(ctx, fileBytes, mimeType)
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

func (s *NoteService) CreateTextNote(userID int, title, text string) (*model.Note, error) {
	filename := title
	if filename == "" {
		filename = "Untitled note"
	}
	note, err := s.repo.CreateTextNote(userID, filename, text)
	if err != nil {
		return nil, err
	}

	go s.ProcessNote(context.Background(), userID, note.ID) // fire-and-forget, don't block the upload response

	return note, nil
}

func (s *NoteService) ListNotes(userID int) ([]model.Note, error) {
	return s.repo.ListByUser(userID)
}

func (s *NoteService) GetDownloadURL(ctx context.Context, userID, noteID int) (string, error) {
	note, err := s.repo.GetByID(userID, noteID)
	if err != nil {
		return "", err
	}
	if note == nil {
		return "", fmt.Errorf("note not found")
	}
	return s.r2.PresignedGetURL(ctx, *note.R2Key, 15*time.Minute)
}

func (s *NoteService) DeleteNote(ctx context.Context, userID, noteID int) error {
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

func (s *NoteService) ProcessNote(ctx context.Context, userID, noteID int) error {
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

	type chunkResult struct {
		content   string
		match     *chunkMatch
		embedding []float32
		err       error
	}

	results := make([]chunkResult, len(chunks))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4) // limit concurrency to 4 at a time, avoid hammering the API

	for i, chunk := range chunks {
		wg.Add(1)
		go func(idx int, c string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			match, err := matchChunkToVerse(ctx, c, bookList)
			if err != nil {
				results[idx] = chunkResult{content: c, err: err}
				return
			}
			embedding, err := embedChunk(ctx, c)
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
		if err := s.chunkRepo.Create(userID, noteID, r.content, r.embedding, r.match.BookID, r.match.Chapter, r.match.VerseStart, r.match.VerseEnd); err != nil {
			log.Println("ProcessNote: insert failed for chunk", i, ":", err)
			s.repo.UpdateStatus(noteID, "failed")
			return err
		}
	}

	log.Println("ProcessNote: done, status -> ready")
	return s.repo.UpdateStatus(noteID, "ready")
}

func (s *NoteService) GetNote(userID, noteID int) (*model.Note, error) {
	return s.repo.GetByID(userID, noteID)
}
