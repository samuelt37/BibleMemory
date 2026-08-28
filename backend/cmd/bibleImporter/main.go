package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/samuelt37/BibleMemory/internal/database"
	"github.com/samuelt37/BibleMemory/internal/importer"
)

type VerseRecord struct {
	Translation string
	Testament   string
	BookOrder   int
	Book        string
	Chapter     int
	Verse       int
	Text        string
}

func main() {
	db, err := database.Connect()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	fmt.Println("Database connected")

	// Ensure table and indexes exist
	schema := `
	CREATE TABLE IF NOT EXISTS bible_verses (
		id SERIAL PRIMARY KEY,
		translation TEXT NOT NULL,
		testament TEXT NOT NULL,
		book_order INT NOT NULL,
		book TEXT NOT NULL,
		chapter INT NOT NULL,
		verse INT NOT NULL,
		text TEXT NOT NULL,
		UNIQUE(translation, book, chapter, verse)
	);

	CREATE INDEX IF NOT EXISTS idx_bible_lookup
	ON bible_verses(translation, book, chapter);

	CREATE INDEX IF NOT EXISTS idx_bible_book_order
	ON bible_verses (translation, book_order);
	`
	if _, err := db.Exec(schema); err != nil {
		panic(fmt.Sprintf("Failed to initialize schema: %v", err))
	}
	fmt.Println("Database schema verified")

	// get list of books for order
	data, err := os.ReadFile("data/kjv/Books.json")
	if err != nil {
		panic(err)
	}
	var bookList []string
	err = json.Unmarshal(data, &bookList)
	if err != nil {
		panic(err)
	}
	bookOrder := make(map[string]int)
	for i, book := range bookList {
		bookOrder[book] = i
	}

	files, err := os.ReadDir("data/kjv")
	if err != nil {
		panic(err)
	}

	fmt.Println("Starting fast batch import...")

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" || file.Name() == "Books.json" {
			continue
		}

		data, err := os.ReadFile("data/kjv/" + file.Name())
		if err != nil {
			panic(err)
		}

		var book importer.Book
		err = json.Unmarshal(data, &book)
		if err != nil {
			panic(err)
		}

		bookNum := bookOrder[book.Book]
		testament := "Old"
		if bookNum >= 39 {
			testament = "New"
		}

		var records []VerseRecord
		for _, chapter := range book.Chapters {
			chapterNum, err := strconv.Atoi(chapter.Chapter)
			if err != nil {
				panic(err)
			}

			for _, verse := range chapter.Verses {
				verseNum, err := strconv.Atoi(verse.Verse)
				if err != nil {
					panic(err)
				}

				records = append(records, VerseRecord{
					Translation: "KJV",
					Testament:   testament,
					BookOrder:   bookNum,
					Book:        book.Book,
					Chapter:     chapterNum,
					Verse:       verseNum,
					Text:        verse.Text,
				})
			}
		}

		// Insert in chunks of 500
		const chunkSize = 500
		for i := 0; i < len(records); i += chunkSize {
			end := i + chunkSize
			if end > len(records) {
				end = len(records)
			}
			chunk := records[i:end]

			valueStrings := make([]string, 0, len(chunk))
			valueArgs := make([]interface{}, 0, len(chunk)*7)
			for j, r := range chunk {
				base := j * 7
				valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d)",
					base+1, base+2, base+3, base+4, base+5, base+6, base+7))
				valueArgs = append(valueArgs, r.Translation, r.Testament, r.BookOrder, r.Book, r.Chapter, r.Verse, r.Text)
			}

			stmt := fmt.Sprintf(
				"INSERT INTO bible_verses (translation, testament, book_order, book, chapter, verse, text) VALUES %s ON CONFLICT (translation, book, chapter, verse) DO NOTHING",
				strings.Join(valueStrings, ","),
			)

			_, err = db.Exec(stmt, valueArgs...)
			if err != nil {
				panic(fmt.Sprintf("Failed inserting chunk for %s: %v", book.Book, err))
			}
		}

		fmt.Println("Imported", book.Book)
	}

	fmt.Println("🎉 All 66 books imported successfully!")
}
