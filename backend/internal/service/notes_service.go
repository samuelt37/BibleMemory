package service

import (
	"context"
	"fmt"
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
	key := fmt.Sprintf("users/%d/%s-%s", userID, uuid.NewString(), header.Filename)
	mimeType := header.Header.Get("Content-Type")

	if err := s.r2.Upload(ctx, key, file, mimeType); err != nil {
		return nil, err
	}

	return s.repo.CreateNote(userID, header.Filename, key, mimeType, header.Size)
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
		log.Println("ProcessNote: note not found", noteID, err)
		return fmt.Errorf("note not found")
	}
	if note.RawText == nil {
		log.Println("ProcessNote: no raw text", noteID)
		return fmt.Errorf("note has no text to process yet")
	}

	chunks := chunkText(*note.RawText, 400)
	log.Println("ProcessNote: chunked into", len(chunks), "pieces")

	bookList, err := s.buildBookList()
	if err != nil {
		log.Println("ProcessNote: buildBookList failed:", err)
		s.repo.UpdateStatus(noteID, "failed")
		return err
	}

	for i, chunk := range chunks {
		match, err := matchChunkToVerse(ctx, chunk, bookList)
		if err != nil {
			log.Println("ProcessNote: matchChunkToVerse failed on chunk", i, ":", err)
			s.repo.UpdateStatus(noteID, "failed")
			return err
		}

		embedding, err := embedChunk(ctx, chunk)
		if err != nil {
			log.Println("ProcessNote: embedChunk failed on chunk", i, ":", err)
			s.repo.UpdateStatus(noteID, "failed")
			return err
		}

		if err := s.chunkRepo.Create(userID, noteID, chunk, embedding, match.BookID, match.Chapter, match.VerseStart, match.VerseEnd); err != nil {
			log.Println("ProcessNote: chunkRepo.Create failed on chunk", i, ":", err)
			s.repo.UpdateStatus(noteID, "failed")
			return err
		}
	}

	log.Println("ProcessNote: done, status -> ready")
	return s.repo.UpdateStatus(noteID, "ready")
}
