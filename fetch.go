package main

import (
	"fmt"
	"sort"
	"time"
)

// https://developer.themoviedb.org/docs/rate-limiting
const requestInterval = time.Second / 50 // Limit to 50 requests per second

func fetchActorCountsForUser(username string, cache map[string][]string) []actorEntry {
	movieSlugs := fetchMovieSlugs(username)
	actorCounts := make(map[string]int)
	var actors []string
	for _, slug := range movieSlugs {
		if cachedActors, found := cache[slug]; found {
			actors = cachedActors
			fmt.Println("Cache hit for slug:", slug)
		} else {
			actors = fetchActorsFromTMDB(slug)
			cache[slug] = actors
			time.Sleep(requestInterval) // Rate limit API calls
		}
		for _, actor := range actors {
			actorCounts[actor]++
		}
	}

	// Filter out actors appearing only once
	for actor, count := range actorCounts {
		if count < 2 {
			delete(actorCounts, actor)
		}
	}

	sortedActors := sortActorCounts(actorCounts)

	return sortedActors
}

type actorEntry struct {
	Name  string
	Count int
}

func sortActorCounts(actorCounts map[string]int) []actorEntry {
	var sortedActors []actorEntry
	for actor, count := range actorCounts {
		sortedActors = append(sortedActors, actorEntry{Name: actor, Count: count})
	}

	sort.Slice(sortedActors, func(i, j int) bool {
		return sortedActors[i].Count > sortedActors[j].Count
	})

	return sortedActors
}
