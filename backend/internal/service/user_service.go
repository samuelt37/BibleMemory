package service

import (
	"github.com/samuelt37/BibleMemory/internal/repository"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) EnsureUser(clerkUserID string) (int, error) {
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
