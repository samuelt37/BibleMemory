package service

import (
	"github.com/samuelt37/BibleMemory/internal/model"
	"github.com/samuelt37/BibleMemory/internal/repository"
)

type SessionService struct {
	repo *repository.SessionRepository
}

func NewSessionService(repo *repository.SessionRepository) *SessionService {
	return &SessionService{repo: repo}
}

func (s *SessionService) SaveSession(userID int, ranges []model.ScriptureRange) (*model.MemorySession, error) {
	return s.repo.Create(userID, ranges, false)
}

func (s *SessionService) GetHistory(userID int) ([]model.MemorySession, error) {
	return s.repo.ListHistory(userID, 20)
}

func (s *SessionService) GetBookmarks(userID int) ([]model.MemorySession, error) {
	return s.repo.ListBookmarked(userID)
}

func (s *SessionService) ToggleBookmark(userID int, sessionID int, bookmarked bool) error {
	return s.repo.SetBookmarked(userID, sessionID, bookmarked)
}

func (s *SessionService) DeleteSession(userID int, sessionID int) error {
	return s.repo.Delete(userID, sessionID)
}
