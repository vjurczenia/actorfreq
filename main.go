package main

import (
	"log/slog"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		slog.Error("Error loading .env file")
		return
	}

	loadCache()

	// cli()
	startServer()
}
