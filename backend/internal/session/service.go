package session

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) SaveSession(userID int, ranges []ScriptureRange) (*Session, error) {
	return s.repo.Create(userID, ranges, false)
}

func (s *Service) GetHistory(userID int) ([]Session, error) {
	return s.repo.ListHistory(userID, 20)
}

func (s *Service) GetBookmarks(userID int) ([]Session, error) {
	return s.repo.ListBookmarked(userID)
}

func (s *Service) ToggleBookmark(userID int, sessionID int, bookmarked bool) error {
	return s.repo.SetBookmarked(userID, sessionID, bookmarked)
}

func (s *Service) DeleteSession(userID int, sessionID int) error {
	return s.repo.Delete(userID, sessionID)
}
