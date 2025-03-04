package main

import (
	"net/http"
	"sort"
	"sync"
)

type FilmDetails struct {
	Title string
	Cast  []string
}

type actorDetails struct {
	Name   string
	Movies []movieDetails
}

type movieDetails struct {
	FilmSlug string
	Title    string
}

func fetchActors(username string, sortStrategy string, lastNMovies int, w *http.ResponseWriter) []actorDetails {
	filmSlugs := fetchFilmSlugs(username, sortStrategy)

	if lastNMovies > 0 && lastNMovies < len(filmSlugs) {
		filmSlugs = filmSlugs[:lastNMovies]
	}

	if w != nil {
		sendMapAsSSEData(*w, map[string]int{
			"total": len(filmSlugs),
		})
	}

	var wg sync.WaitGroup
	actors := make(map[string]*actorDetails)
	var actorsMutex sync.Mutex
	for _, slug := range filmSlugs {
		wg.Add(1)
		go func(slug string) {
			defer wg.Done()

			cacheMutex.Lock()
			cachedData, found := cache[slug]
			cacheMutex.Unlock()

			if !found {
				cachedData = fetchFilmDetails(slug)
			}

			actorsMutex.Lock()
			for _, actorName := range cachedData.Cast {
				actor, found := actors[actorName]
				if !found {
					actors[actorName] = &actorDetails{Name: actorName}
					actor = actors[actorName]
				}
				actor.Movies = append(actor.Movies, movieDetails{FilmSlug: slug, Title: cachedData.Title})
			}
			actorsMutex.Unlock()

			if w != nil {
				sendMapAsSSEData(*w, map[string]int{
					"progress": 1,
				})
			}
		}(slug)
	}

	wg.Wait()

	cleanedActors := cleanActors(actors)

	saveCache()

	return cleanedActors
}

func cleanActors(actors map[string]*actorDetails) []actorDetails {
	// Filter out actors appearing only once and sort by movies descending
	cleanedActors := []actorDetails{}
	for _, actor := range actors {
		if len(actor.Movies) > 1 {
			cleanedActors = append(cleanedActors, *actor)
		}
	}

	sort.Slice(cleanedActors, func(i, j int) bool {
		return len(cleanedActors[i].Movies) > len(cleanedActors[j].Movies)
	})

	return cleanedActors
}
