package main

import (
	"flag"
	"fmt"
	"os"
)

func cli() {
	// Parse command-line arguments
	username := flag.String("username", "", "The username to fetch data for")
	flag.Parse()

	// Ensure a username was provided
	if *username == "" {
		fmt.Println("Error: Username must be provided.")
		return
	}

	// Fetch the TMDB API key from environment variables
	tmdbAPIKey := os.Getenv("TMDB_API_KEY")
	if tmdbAPIKey == "" {
		fmt.Println("TMDB API key is missing from the .env file.")
		return
	}

	cache := loadCache()
	actorCounts := fetchActorsForUser(*username, tmdbAPIKey, cache)
	saveCache(cache)

	// Output top 10 actors
	printTopActors(actorCounts)
}

func printTopActors(actorCounts map[string]int) {
	sortedActors := sortActorCounts(actorCounts)

	fmt.Println("Top 10 Actor appearance counts:")
	for i, entry := range sortedActors {
		if i == 0 && entry.Count == 1 {
			fmt.Printf("Actorigami!")
		}
		if i >= 10 {
			break
		}
		fmt.Printf("%s: %d\n", entry.Name, entry.Count)
	}
}
