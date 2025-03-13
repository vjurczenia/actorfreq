package main

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/vjurczenia/actorfreq/actorfreq"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		slog.Warn("No .env file found")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	actorfreq.SetUpDB()

	// actorfreq.CLI()
	actorfreq.StartServer()
}
