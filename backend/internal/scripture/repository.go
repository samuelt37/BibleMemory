package scripture

import (
	"database/sql"
	"fmt"
	"strings"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) GetBooks() ([]BookInfo, error) {
	query := `
		SELECT MIN(book_order) AS id, book, MAX(chapter) AS chapters
		FROM bible_verses
		WHERE translation = 'KJV'
		GROUP BY book
		ORDER BY MIN(book_order)
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []BookInfo

	for rows.Next() {
		var b BookInfo
		if err := rows.Scan(&b.ID, &b.Book, &b.Chapters); err != nil {
			return nil, err
		}
		books = append(books, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return books, nil
}

func (r *Repository) GetChapters(book string) (int, error) {
	query := `
		SELECT COUNT(DISTINCT chapter)
		FROM bible_verses
		WHERE translation = 'KJV'
		AND book = $1
	`
	var count int

	err := r.db.QueryRow(query, book).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *Repository) GetScripture(queryParams Query) ([]VerseInfo, error) {
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

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var verses []VerseInfo

	for rows.Next() {
		var verse VerseInfo
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
	start Reference,
	end Reference,
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
	ref Reference,
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
	ref Reference,
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
