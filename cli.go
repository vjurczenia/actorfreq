package main

import (
	"flag"
	"fmt"
	"log/slog"
)

func cli() {
	// Parse command-line arguments
	username := flag.String("username", "", "The username to fetch data for")
	lastNMovies := flag.Int("lastNMovies", -1, "Last N movies to fetch data for")
	flag.Parse()

	// Ensure a username was provided
	if *username == "" {
		slog.Error("Error: Username must be provided.")
		return
	}

	actors := fetchActors(*username, *lastNMovies, nil)

	// Output top 10 actors
	printTopActors(actors)
}

func printTopActors(actors []actorDetails) {
	fmt.Println("Top 10 Actor appearance counts:")
	for i, entry := range actors {
		if i == 0 && len(entry.Movies) == 1 {
			fmt.Printf("Actorigami!")
		}
		if i >= 10 {
			break
		}
		fmt.Printf("%s: %d\n", entry.Name, len(entry.Movies))
	}
}
