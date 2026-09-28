package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/samuelt37/BibleMemory/internal/database"
	"github.com/samuelt37/BibleMemory/internal/notes"
	"github.com/samuelt37/BibleMemory/internal/router"
	"github.com/samuelt37/BibleMemory/internal/scripture"
	"github.com/samuelt37/BibleMemory/internal/session"
	"github.com/samuelt37/BibleMemory/internal/storage"
	"github.com/samuelt37/BibleMemory/internal/summary"
	"github.com/samuelt37/BibleMemory/internal/user"
)

func main() {
	clerkKey := os.Getenv("CLERK_SECRET_KEY")
	if clerkKey == "" {
		fmt.Println("⚠️  WARNING: CLERK_SECRET_KEY environment variable is not set! Clerk auth will fail.")
	} else {
		clerk.SetKey(clerkKey)
	}
	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	db, err := database.Connect()
	if err != nil {
		panic(err)
	}

	defer db.Close()

	fmt.Println("Database connected")

	scriptureRepo := scripture.NewRepository(db)
	scriptureService := scripture.NewService(scriptureRepo)
	scriptureHandler := scripture.NewHandler(scriptureService)

	userRepo := user.NewRepository(db)
	userService := user.NewService(userRepo)

	sessionRepo := session.NewRepository(db)
	sessionService := session.NewService(sessionRepo)
	sessionHandler := session.NewHandler(sessionService)

	r2Client, err := storage.NewR2Client(context.Background())
	if err != nil {
		panic(err)
	}

	noteRepo := notes.NewRepository(db)
	noteChunkRepo := notes.NewChunkRepository(db)
	noteService := notes.NewService(noteRepo, noteChunkRepo, scriptureRepo, r2Client)
	noteHandler := notes.NewHandler(noteService)

	summaryService := summary.NewService(scriptureRepo, noteChunkRepo)
	summaryHandler := summary.NewHandler(summaryService)

	r := router.NewRouter(scriptureHandler, summaryHandler, sessionHandler, noteHandler, userService)

	fmt.Println("Server running on :" + port)

	err = http.ListenAndServe(":"+port, r)
	if err != nil {
		panic(err)
	}
}
