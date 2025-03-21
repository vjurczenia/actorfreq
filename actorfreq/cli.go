package actorfreq

import (
	"flag"
	"fmt"
	"log/slog"
)

func CLI() {
	// Parse command-line arguments
	username := flag.String("username", "", "The username to fetch data for")
	sortStrategy := flag.String("sortStrategy", "", "How to sort movies")
	topNMovies := flag.Int("topNMovies", -1, "Last N movies to fetch data for")
	flag.Parse()

	// Ensure a username was provided
	if *username == "" {
		slog.Error("Error: Username must be provided.")
		return
	}

	rc := requestConfig{
		sortStrategy: *sortStrategy,
		topNMovies:   *topNMovies,
	}

	actors := fetchActors(*username, rc, nil)

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
