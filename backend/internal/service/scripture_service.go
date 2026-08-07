package service

import (
	"github.com/samuelt37/BibleMemory/internal/model"
	"github.com/samuelt37/BibleMemory/internal/repository"
	"github.com/samuelt37/BibleMemory/internal/dto"
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

func (s *ScriptureService) GetScripture(
	query dto.ScriptureQuery,
) ([]model.Verse, error) {
	if query.VerseStart != nil && query.VerseEnd == nil {
		query.VerseEnd = nil
	}
	return s.repo.GetScripture(query)
}
