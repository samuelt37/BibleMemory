package service

import (
	"github.com/samuelt37/BibleMemory/internal/model"
	"github.com/samuelt37/BibleMemory/internal/repository"
)

type ScriptureService struct {
	repo *repository.ScriptureRepository
}

func NewScriptureService(
	repo *repository.ScriptureRepository,
) *ScriptureService {

	return &ScriptureService{
		repo: repo,
	}
}

func (s *ScriptureService) GetBook(
	translation string,
	book string,
) ([]model.Verse, error) {

	return s.repo.GetBook(
		translation,
		book,
	)
}
