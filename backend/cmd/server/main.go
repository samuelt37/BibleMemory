package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/samuelt37/BibleMemory/internal/database"
	"github.com/samuelt37/BibleMemory/internal/handler"
	"github.com/samuelt37/BibleMemory/internal/repository"
	"github.com/samuelt37/BibleMemory/internal/router"
	"github.com/samuelt37/BibleMemory/internal/service"
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

	scriptureRepo := repository.NewScriptureRepository(db)
	scriptureService := service.NewScriptureService(scriptureRepo)
	scriptureHandler := handler.NewScriptureHandler(scriptureService)

	summaryService := service.NewSummaryService(scriptureRepo)
	summaryHandler := handler.NewSummaryHandler(summaryService)

	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)

	sessionRepo := repository.NewSessionRepository(db)
	sessionService := service.NewSessionService(sessionRepo)
	sessionHandler := handler.NewSessionHandler(sessionService)

	r := router.NewRouter(scriptureHandler, sessionHandler, userService)
	router.RegisterScriptureRoutes(r, scriptureHandler)
	router.RegisterSummaryRoutes(r, summaryHandler)

	fmt.Println("Server running on :" + port)

	err = http.ListenAndServe(":"+port, r)
	if err != nil {
		panic(err)
	}
}
