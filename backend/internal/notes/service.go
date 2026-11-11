package notes

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"path"
	"path/filepath"
	"sort"
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

func (s *Service) UploadNote(ctx context.Context, userID int, file multipart.File, header *multipart.FileHeader, mimeType string) (*Note, error) {
	key := fmt.Sprintf("users/%d/%s-%s", userID, uuid.NewString(), path.Base(header.Filename))
	isVideo := strings.HasPrefix(mimeType, "video/")

	var fileBytes []byte
	var body io.Reader = file
	if !isVideo {
		b, err := io.ReadAll(file)
		if err != nil {
			return nil, err
		}
		fileBytes, body = b, bytes.NewReader(b)
	}

	if err := s.r2.Upload(ctx, key, body, mimeType); err != nil {
		return nil, err
	}

	note, err := s.repo.CreateNote(userID, header.Filename, key, mimeType, header.Size)
	if err != nil {
		return nil, err
	}

	if isVideo {
		go s.processVideo(userID, note.ID, key, header.Size, mimeType)
	} else {
		go s.extractAndProcess(context.Background(), userID, note.ID, fileBytes, mimeType)
	}
	return note, nil
}

func (s *Service) processVideo(userID, noteID int, key string, size int64, mimeType string) {
	log.Println("processVideo: starting for note", noteID, "size:", size, "bytes")

	extractCtx, cancelExtract := context.WithTimeout(context.Background(), 40*time.Minute)
	defer cancelExtract()

	obj, err := s.r2.Download(extractCtx, key)
	if err != nil {
		log.Println("processVideo: r2 download failed for note", noteID, ":", err)
		s.repo.UpdateStatus(noteID, "failed")
		return
	}
	defer obj.Close()

	f, err := s.gemini.UploadFile(extractCtx, obj, size, mimeType, key)
	if err != nil {
		log.Println("processVideo: gemini upload failed for note", noteID, ":", err)
		s.repo.UpdateStatus(noteID, "failed")
		return
	}
	defer s.gemini.DeleteFile(context.Background(), f.Name)

	text, err := extractTextFromVideo(extractCtx, s.gemini, f)
	if err != nil {
		log.Println("processVideo: extraction failed for note", noteID, ":", err)
		s.repo.UpdateStatus(noteID, "failed")
		return
	}
	clean, segs := parseTimestamped(text)

	if err := s.repo.UpdateRawText(noteID, clean); err != nil {
		log.Println("processVideo: failed to save extracted text for note", noteID, ":", err)
		s.repo.UpdateStatus(noteID, "failed")
		return
	}

	if err := s.repo.SaveSegments(noteID, segs); err != nil {
		log.Println("processVideo: failed to save segments for note", noteID, ":", err)
	}

	// Fresh budget for chunk -> tag -> embed, so a slow upload can't starve it
	processCtx, cancelProcess := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancelProcess()

	if err := s.ProcessNote(processCtx, userID, noteID); err != nil {
		log.Println("processVideo: ProcessNote failed for note", noteID, ":", err)
	}
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

	segs, _ := s.repo.GetSegments(noteID)
	starts := make([]*int, len(chunks))
	if len(segs) > 0 {
		from := 0
		for i, c := range chunks {
			if idx := findFrom(*note.RawText, from, c); idx >= 0 {
				starts[i] = secondsAt(segs, idx)
				from = idx + 1
			} else if i > 0 {
				starts[i] = starts[i-1]
			}
		}
	}

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
				// rate limits and cancellations still fail the note; a bad response doesn't
				if errors.Is(err, ErrRateLimited) || ctx.Err() != nil {
					results[idx] = chunkResult{content: c, err: err}
					return
				}
				log.Printf("ProcessNote: chunk %d tagging failed, saving untagged: %v", idx, err)
				match = &chunkMatch{} // no refs; Retag can fill it in later
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
		if err := s.chunkRepo.Create(userID, noteID, r.content, r.embedding, validRefs(r.match.Refs), starts[i]); err != nil {
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
		return ErrNotFound
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
			if attempt == 0 && ctx.Err() == nil {
				continue // any other failure: try once more
			}
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

func (s *Service) CreateChunk(ctx context.Context, userID, noteID int, in CreateChunkInput) (*Chunk, error) {
	note, err := s.repo.GetByID(userID, noteID)
	if err != nil {
		return nil, err
	}
	if note == nil {
		return nil, ErrNotFound
	}

	embedding, err := s.gemini.EmbedText(ctx, in.Content, 768)
	if err != nil {
		return nil, err
	}

	var start *int
	if note.RawText != nil {
		if idx := findFrom(*note.RawText, 0, in.Content); idx >= 0 {
			segs, _ := s.repo.GetSegments(noteID)
			start = secondsAt(segs, idx)
		}
	}

	chunkID, err := s.chunkRepo.CreateWithRefs(ctx, userID, noteID, in.Content, embedding, in.Refs, start)
	if err != nil {
		return nil, err
	}

	return &Chunk{ID: chunkID, Content: in.Content, Refs: in.Refs, StartSeconds: start}, nil
}

func (s *Service) DeleteReference(ctx context.Context, userID, noteID int, ref verseRef) ([]DeletedRef, error) {
	note, err := s.repo.GetByID(userID, noteID)
	if err != nil {
		return nil, err
	}
	if note == nil {
		return nil, ErrNotFound
	}

	return s.chunkRepo.DeleteRefs(ctx, userID, noteID, ref)
}

func (s *Service) NoteCountsByBook(ctx context.Context, userID int) (map[int]int, error) {
	rows, err := s.chunkRepo.CountByBook(ctx, userID)
	if err != nil {
		return nil, err
	}
	counts := make(map[int]int, len(rows))
	for _, r := range rows {
		counts[r.BookID] = r.Count
	}
	return counts, nil
}

const maxPresignedVideoBytes = 1 << 30 // 1GB; Gemini's Files API tops out around 2GB per file

func isVideoExtension(ext string) (string, bool) {
	switch strings.ToLower(ext) {
	case ".mp4", ".m4v":
		return "video/mp4", true
	case ".mov":
		return "video/quicktime", true
	case ".webm":
		return "video/webm", true
	case ".mkv":
		return "video/x-matroska", true
	case ".avi":
		return "video/x-msvideo", true
	case ".wmv":
		return "video/x-ms-wmv", true
	case ".3gp":
		return "video/3gpp", true
	default:
		return "", false
	}
}

func resolveVideoMimeType(filename, mimeType string) string {
	if strings.HasPrefix(mimeType, "video/") {
		return mimeType
	}
	if videoMime, ok := isVideoExtension(filepath.Ext(filename)); ok {
		return videoMime
	}
	if extType := mime.TypeByExtension(filepath.Ext(filename)); extType != "" {
		extType, _, _ = strings.Cut(extType, ";")
		if strings.HasPrefix(extType, "video/") {
			return extType
		}
	}
	return mimeType
}

func (s *Service) PresignVideoUpload(ctx context.Context, userID int, filename, contentType string, size int64) (url, key string, err error) {
	key = fmt.Sprintf("users/%d/%s-%s", userID, uuid.NewString(), path.Base(filename))
	url, err = s.r2.PresignedPutURL(ctx, key, contentType, 30*time.Minute)
	return url, key, err
}

func (s *Service) CompleteVideoUpload(ctx context.Context, userID int, key, filename string) (*Note, error) {
	// the key must belong to this user, otherwise anyone could claim anyone's object
	if !strings.HasPrefix(key, fmt.Sprintf("users/%d/", userID)) {
		return nil, errors.New("invalid key")
	}

	// don't trust the client: check what actually landed in R2
	size, mimeType, err := s.r2.Head(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("object not found: %w", err)
	}

	mimeType = resolveVideoMimeType(filename, mimeType)
	if !strings.HasPrefix(mimeType, "video/") || size <= 0 || size > maxPresignedVideoBytes {
		_ = s.r2.Delete(ctx, key)
		return nil, errors.New("invalid video")
	}

	note, err := s.repo.CreateNote(userID, filename, key, mimeType, size)
	if err != nil {
		return nil, err
	}

	log.Printf("CompleteVideoUpload: verified in R2 for note %q (noteID: %d, size: %d, mime: %q)", filename, note.ID, size, mimeType)
	go s.processVideo(userID, note.ID, key, size, mimeType)
	return note, nil
}

const videoPartSize = 16 << 20 // R2 needs parts of equal size (except the last) and >= 5MiB

type VideoMultipart struct {
	Key      string   `json:"key"`
	UploadID string   `json:"uploadId"`
	PartSize int64    `json:"partSize"`
	URLs     []string `json:"urls"`
}

func (s *Service) StartVideoMultipart(ctx context.Context, userID int, filename, contentType string, size int64) (*VideoMultipart, error) {
	contentType = resolveVideoMimeType(filename, contentType)
	if !strings.HasPrefix(contentType, "video/") || size <= 0 || size > maxPresignedVideoBytes {
		return nil, errors.New("invalid video")
	}

	key := fmt.Sprintf("users/%d/%s-%s", userID, uuid.NewString(), path.Base(filename))
	uploadID, err := s.r2.CreateMultipartUpload(ctx, key, contentType)
	if err != nil {
		return nil, err
	}

	n := int((size + videoPartSize - 1) / videoPartSize)
	urls := make([]string, n)
	for i := range urls {
		u, err := s.r2.PresignUploadPart(ctx, key, uploadID, int32(i+1), 2*time.Hour)
		if err != nil {
			s.r2.AbortMultipartUpload(ctx, key, uploadID)
			return nil, err
		}
		urls[i] = u
	}
	return &VideoMultipart{Key: key, UploadID: uploadID, PartSize: videoPartSize, URLs: urls}, nil
}

func (s *Service) FinishVideoMultipart(ctx context.Context, userID int, key, uploadID, filename string, parts []storage.CompletedPart) (*Note, error) {
	if !strings.HasPrefix(key, fmt.Sprintf("users/%d/", userID)) {
		return nil, errors.New("invalid key")
	}
	sort.Slice(parts, func(i, j int) bool { return parts[i].PartNumber < parts[j].PartNumber })

	if err := s.r2.CompleteMultipartUpload(ctx, key, uploadID, parts); err != nil {
		return nil, fmt.Errorf("complete multipart: %w", err)
	}
	// from here it's the same as before: Head check, CreateNote, go processVideo
	return s.CompleteVideoUpload(ctx, userID, key, filename)
}

// GetMedia returns a playable URL for video/image notes, or nil (no error) for anything else.
func (s *Service) GetMedia(ctx context.Context, userID, noteID int) (*NoteMedia, error) {
	note, err := s.repo.GetByID(userID, noteID)
	if err != nil {
		return nil, err
	}
	if note == nil {
		return nil, ErrNotFound
	}
	if note.R2Key == nil || note.MimeType == nil {
		return nil, nil // text notes
	}

	var mediaType string
	switch {
	case strings.HasPrefix(*note.MimeType, "video/"):
		mediaType = "video"
	case strings.HasPrefix(*note.MimeType, "image/"):
		mediaType = "image"
	default:
		return nil, nil // pdf, plain text, etc.
	}

	url, err := s.r2.PresignedGetURL(ctx, *note.R2Key, 2*time.Hour)
	if err != nil {
		return nil, err
	}
	return &NoteMedia{Type: mediaType, URL: url}, nil
}
