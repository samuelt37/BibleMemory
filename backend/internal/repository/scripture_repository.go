package repository

import (
	"database/sql"

	"github.com/samuelt37/BibleMemory/internal/model"
)

type ScriptureRepository struct {
	db *sql.DB
}

func NewScriptureRepository(db *sql.DB) *ScriptureRepository {
	return &ScriptureRepository{
		db: db,
	}
}

func (r *ScriptureRepository) GetBook(
	translation string,
	book string,
) ([]model.Verse, error) {

	rows, err := r.db.Query(
		`
		SELECT verse, text
		FROM bible_verses
		WHERE translation = $1
		AND book = $2
		ORDER BY chapter, verse
		`,
		translation,
		book,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var verses []model.Verse

	for rows.Next() {
		var verse model.Verse

		err := rows.Scan(
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
