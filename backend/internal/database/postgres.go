package database

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func Connect() (*sql.DB, error) {
	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is not set")
	}

	db, err := sql.Open("postgres", connString)

	for i := 0; i < 10; i++ {
		err = db.Ping()
		if err == nil {
			return db, nil
		}

		fmt.Println("Waiting for database...")
		time.Sleep(time.Second)
	}

	return nil, err
}
