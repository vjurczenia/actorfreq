package main

import (
	"log/slog"
	"sort"
)

func fetchActorCounts(username string, lastNMovies int) []actorEntry {
	filmSlugs := fetchFilmSlugs(username)

	if lastNMovies > 0 && lastNMovies < len(filmSlugs) {
		filmSlugs = filmSlugs[:lastNMovies]
	}

	actorCounts := make(map[string]int)
	var actors []string
	for _, slug := range filmSlugs {
		if cachedActors, found := cache[slug]; found {
			slog.Info("Cache hit", "slug", slug)
			actors = cachedActors
		} else {
			slog.Info("Cache miss", "slug", slug)
			actors = fetchActors(slug)
			if actors != nil {
				cache[slug] = actors
			}
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

	saveCache()

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
