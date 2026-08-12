package service

import (
	"errors"
	"fmt"

	"github.com/samuelt37/BibleMemory/internal/dto"
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

func (s *ScriptureService) GetBooks() ([]string, error) {
	return s.repo.GetBooks()
}

func (s *ScriptureService) GetChapters(
	book string,
) (int, error) {
	return s.repo.GetChapters(book)
}

func (s *ScriptureService) GetScripture(
	query dto.ScriptureQuery,
) ([]model.Verse, error) {
	if len(query.Ranges) == 0 {
		return nil, errors.New("Must have at least a book")
	}

	for i := range query.Ranges {
		r := &query.Ranges[i]

		if err := validateReference(r.Start); err != nil {
			return nil, err
		}

		if r.End != nil {
			if err := validateReference(*r.End); err != nil {
				return nil, err
			}

			if r.Start.Book > r.End.Book {
				return nil, errors.New("book range invalid")
			}

			if (r.Start.Chapter == nil) != (r.End.Chapter == nil) {
				return nil, errors.New("chapter range invalid")
			}

			if (r.Start.Verse == nil) != (r.End.Verse == nil) {
				return nil, errors.New("verse range invalid")
			}

			if r.Start.Book == r.End.Book {
				if r.Start.Chapter != nil && r.End.Chapter != nil {
					if *r.Start.Chapter > *r.End.Chapter {
						return nil, errors.New("chapter range invalid")
					}

					if *r.Start.Chapter == *r.End.Chapter {
						if r.Start.Verse != nil && r.End.Verse != nil {
							if *r.Start.Verse > *r.End.Verse {
								return nil, errors.New("verse range invalid")
							}
						}
					}
				}
			}
		}

		if r.End == nil {
			r.End = &r.Start
		}
	}
	return s.repo.GetScripture(query)
}

func validateReference(ref dto.Reference) error {
	if ref.Book <= 0 {
		return errors.New("book is required")
	}
	if err := validatePositive("chapter", ref.Chapter); err != nil {
		return err
	}
	if err := validatePositive("verse", ref.Verse); err != nil {
		return err
	}
	if ref.Chapter == nil && ref.Verse != nil {
		return errors.New("Cannot have verse without chapter")
	}
	return nil
}

func validatePositive(name string, value *int) error {
	if value != nil && *value <= 0 {
		return fmt.Errorf("%s must be greater than 0", name)
	}
	return nil
}
