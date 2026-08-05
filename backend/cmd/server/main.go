package main

import (
	"fmt"
	"net/http"
	"os"
	
	"github.com/samuelt37/GoStarter/internal/router"
	"github.com/samuelt37/GoStarter/internal/database"
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

	r := router.NewRouter()

	fmt.Println("Server running on :"+port)

	err = http.ListenAndServe(":"+port, r)
	if err != nil {
		panic(err)
	}
}
