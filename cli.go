package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/joho/godotenv"
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

	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
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
	type actorEntry struct {
		Name  string
		Count int
	}

	var sortedActors []actorEntry
	for actor, count := range actorCounts {
		sortedActors = append(sortedActors, actorEntry{Name: actor, Count: count})
	}

	sort.Slice(sortedActors, func(i, j int) bool {
		return sortedActors[i].Count > sortedActors[j].Count
	})

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
