package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081" // Default to 8081 to avoid conflict with API on 8080
	}

	// Serve static files from ui/dist
	// Assumes the binary is run from the project root where "ui/dist" exists
	fs := http.FileServer(http.Dir("./ui/dist"))
	http.Handle("/", fs)

	log.Printf("Web UI server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
