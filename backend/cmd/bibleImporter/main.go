package main

import (
	"fmt"
	"os"
	"encoding/json"
	"strconv"
	"path/filepath"

	"github.com/samuelt37/BibleMemory/internal/database"
	"github.com/samuelt37/BibleMemory/internal/models"
)

func main() {
	db, err := database.Connect()
	if err != nil {
		panic(err)
	}

	defer db.Close()

	fmt.Println("Database connected")

 	files, err := os.ReadDir("data/kjv")
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

    	if filepath.Ext(file.Name()) != ".json" {
        	continue
    	}

		if file.Name() == "Books.json" {
		    continue
		}

		data, err := os.ReadFile("data/kjv/" + file.Name())
		if err != nil {
    		panic(err)
		}

		var book models.Book
		
		err = json.Unmarshal(data, &book)
		if err != nil {
			panic(err)
		}

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

				_, err = db.Exec(
   					`
    				INSERT INTO bible_verses 
   					(translation, book, chapter, verse, text)
    				VALUES ($1, $2, $3, $4, $5)
    				ON CONFLICT (translation, book, chapter, verse) DO NOTHING
    				`,
    				"KJV",
    				book.Book,
    				chapterNum,
    				verseNum,
    				verse.Text,
				)

				if err != nil {
    				panic(err)
				}
			}
		}
		fmt.Println("Imported", book.Book)
	}
}
