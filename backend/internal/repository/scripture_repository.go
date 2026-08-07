package repository

import (
	"database/sql"
	"fmt"

	"github.com/samuelt37/BibleMemory/internal/model"
	"github.com/samuelt37/BibleMemory/internal/dto"
)

type ScriptureRepository struct {
	db *sql.DB
}

func NewScriptureRepository(db *sql.DB) *ScriptureRepository {
	return &ScriptureRepository {
		db: db,
	}
}

func (r *ScriptureRepository) GetScripture(
	queryParams dto.ScriptureQuery,
) ([]model.Verse, error) {
	
	query := `
		SELECT chapter, verse, text
    	FROM bible_verses
    	WHERE translation = $1
    	AND book = $2
	`
	args := []interface{}{
		queryParams.Translation,
		queryParams.Book,
	}

	paramCount := 2
	if queryParams.Chapter != nil {
		paramCount++
		query += fmt.Sprintf(" AND chapter = $%d", paramCount)
		args = append(args, *queryParams.Chapter)
		
		if queryParams.Verse != nil {
			paramCount ++
			query += fmt.Sprintf(" AND verse = $%d", paramCount)
			args = append(args, *queryParams.Verse)	
		} else if queryParams.VerseStart != nil && queryParams.VerseEnd != nil {
			paramCount++
			query += fmt.Sprintf(
				" AND verse BETWEEN $%d AND $%d",
				paramCount,
				paramCount+1,
			)
			args = append(args, *queryParams.VerseStart, *queryParams.VerseEnd)
			paramCount += 2
		}
	}
	query += " ORDER BY chapter, verse"
	fmt.Println(query)
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var verses []model.Verse

	for rows.Next() {
		var verse model.Verse

		err := rows.Scan(
			&verse.Chapter,
			&verse.Verse,
			&verse.Text,
		)

		if err != nil {
			return nil, err
		}

		verses = append(verses, verse)
	}

	return verses, nil
}
