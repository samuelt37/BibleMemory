package user

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) EnsureUser(clerkUserID string) (int, error) {
	existing, err := s.repo.GetByClerkID(clerkUserID)
	if err != nil {
		return 0, err
	}
	if existing != nil {
		return existing.ID, nil
	}

	created, err := s.repo.Create(clerkUserID)
	if err != nil {
		return 0, err
	}
	return created.ID, nil
}
