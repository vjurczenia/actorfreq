package main

import (
	"flag"
	"fmt"
	"log/slog"
)

func cli() {
	// Parse command-line arguments
	username := flag.String("username", "", "The username to fetch data for")
	flag.Parse()

	// Ensure a username was provided
	if *username == "" {
		slog.Error("Error: Username must be provided.")
		return
	}

	actorCounts := fetchActorCounts(*username)

	// Output top 10 actors
	printTopActors(actorCounts)
}

func printTopActors(actorCounts []actorEntry) {
	fmt.Println("Top 10 Actor appearance counts:")
	for i, entry := range actorCounts {
		if i == 0 && entry.Count == 1 {
			fmt.Printf("Actorigami!")
		}
		if i >= 10 {
			break
		}
		fmt.Printf("%s: %d\n", entry.Name, entry.Count)
	}
}
