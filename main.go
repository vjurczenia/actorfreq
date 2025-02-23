package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/joho/godotenv"
)

func main() {
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

	// Load cached data from file if it exists
	cache := loadCache()
	actorCounts := make(map[string]int)
	page := 1
	for {
		// Use the user name and page number in the URL
		url := fmt.Sprintf("https://letterboxd.com/%s/films/by/date/page/%d", *username, page)
		fmt.Println("Fetching:", url)

		actors := fetchActorsForPage(url, tmdbAPIKey, cache)
		if len(actors) == 0 {
			fmt.Println("No more actors found for page:", page)
			break // Exit loop when no actors are found
		}
		for _, actor := range actors {
			actorCounts[actor]++
		}

		page++
	}

	// Save cache to file
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
		if i >= 10 {
			break
		}
		fmt.Printf("%s: %d\n", entry.Name, entry.Count)
	}
}
