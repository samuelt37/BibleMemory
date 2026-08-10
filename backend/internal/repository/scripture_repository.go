package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/samuelt37/BibleMemory/internal/dto"
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

func (r *ScriptureRepository) GetScripture(
	queryParams dto.ScriptureQuery,
) ([]model.Verse, error) {

	query := `
		SELECT book, chapter, verse, text
		FROM bible_verses
		WHERE translation = $1
		AND (
	`

	args := []interface{}{
		queryParams.Translation,
	}

	paramCount := 1
	var conditions []string

	for _, scriptureRange := range queryParams.Ranges {
		conditions = append(conditions, buildRangeCondition(scriptureRange.Start, *scriptureRange.End, &paramCount, &args))
	}

	query += strings.Join(conditions, " OR ")

	query += `
		)
		ORDER BY book_order, chapter, verse
	`
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
			&verse.Book,
			&verse.Chapter,
			&verse.Verse,
			&verse.Text,
		)

		if err != nil {
			return nil, err
		}

		verses = append(verses, verse)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return verses, nil
}

func buildRangeCondition(
	start dto.Reference,
	end dto.Reference,
	paramCount *int,
	args *[]interface{},
) string {

	startSQL := buildStartCondition(start, paramCount, args)
	endSQL := buildEndCondition(end, paramCount, args)

	return fmt.Sprintf(
		"(%s) AND (%s)",
		startSQL,
		endSQL,
	)
}

func buildStartCondition(
	ref dto.Reference,
	paramCount *int,
	args *[]interface{},
) string {

	*paramCount++
	bookParam := *paramCount
	*args = append(*args, ref.Book)

	if ref.Chapter == nil {
		return fmt.Sprintf(
			"book_order >= $%d",
			bookParam,
		)
	}

	*paramCount++
	chapterParam := *paramCount
	*args = append(*args, *ref.Chapter)

	if ref.Verse == nil {
		return fmt.Sprintf(`
			(
				book_order > $%d
				OR (book_order = $%d AND chapter >= $%d)
			)
		`,
			bookParam,
			bookParam,
			chapterParam,
		)
	}

	*paramCount++
	verseParam := *paramCount
	*args = append(*args, *ref.Verse)

	return fmt.Sprintf(`
		(
			book_order > $%d
			OR (
				book_order = $%d
				AND chapter > $%d
			)
			OR (
				book_order = $%d
				AND chapter = $%d
				AND verse >= $%d
			)
		)
	`,
		bookParam,
		bookParam,
		chapterParam,
		bookParam,
		chapterParam,
		verseParam,
	)
}

func buildEndCondition(
	ref dto.Reference,
	paramCount *int,
	args *[]interface{},
) string {

	*paramCount++
	bookParam := *paramCount
	*args = append(*args, ref.Book)

	if ref.Chapter == nil {
		return fmt.Sprintf(
			"book_order <= $%d",
			bookParam,
		)
	}

	*paramCount++
	chapterParam := *paramCount
	*args = append(*args, *ref.Chapter)

	if ref.Verse == nil {
		return fmt.Sprintf(`
			(
				book_order < $%d
				OR (book_order = $%d AND chapter <= $%d)
			)
		`,
			bookParam,
			bookParam,
			chapterParam,
		)
	}

	*paramCount++
	verseParam := *paramCount
	*args = append(*args, *ref.Verse)

	return fmt.Sprintf(`
		(
			book_order < $%d
			OR (
				book_order = $%d
				AND chapter < $%d
			)
			OR (
				book_order = $%d
				AND chapter = $%d
				AND verse <= $%d
			)
		)
	`,
		bookParam,
		bookParam,
		chapterParam,
		bookParam,
		chapterParam,
		verseParam,
	)
}
