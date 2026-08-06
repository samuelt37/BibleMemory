package main

import (
	"fmt"
	"net/http"
	"os"
	
	"github.com/samuelt37/BibleMemory/internal/router"
	"github.com/samuelt37/BibleMemory/internal/handler"
	"github.com/samuelt37/BibleMemory/internal/service"
	"github.com/samuelt37/BibleMemory/internal/repository"
	"github.com/samuelt37/BibleMemory/internal/database"

)

func main() {
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
	
	r := router.NewRouter()
	router.RegisterScriptureRoutes(r, scriptureHandler)

	fmt.Println("Server running on :"+port)

	err = http.ListenAndServe(":"+port, r)
	if err != nil {
		panic(err)
	}
}
