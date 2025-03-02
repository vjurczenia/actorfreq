package main

import (
	"log/slog"
	"net/http"
	"sort"
)

type FilmDetails struct {
	Title string
	Cast  []string
}

func fetchActorCounts(username string, lastNMovies int, w *http.ResponseWriter) []actorEntry {
	filmSlugs := fetchFilmSlugs(username)

	if lastNMovies > 0 && lastNMovies < len(filmSlugs) {
		filmSlugs = filmSlugs[:lastNMovies]
	}

	if w != nil {
		sendMapAsSSEData(*w, map[string]int{
			"total": len(filmSlugs),
		})
	}

	actorCounts := make(map[string][]string)

	for i, slug := range filmSlugs {
		cacheMutex.Lock()
		cachedData, found := cache[slug]
		cacheMutex.Unlock()

		if found {
			slog.Info("Cache hit", "slug", slug)
			for _, actor := range cachedData.Cast {
				actorCounts[actor] = append(actorCounts[actor], cachedData.Title)
			}
		} else {
			slog.Info("Cache miss", "slug", slug)
			movieResults, err := searchMovie(slug)
			if err != nil || len(movieResults.Results) == 0 {
				slog.Error("Error searching movie", "slug", slug)
				continue
			}

			topMovieResult := movieResults.Results[0]
			castResult, err := fetchMovieCredits(topMovieResult.ID)
			if err != nil {
				slog.Error("Error fetching cast for movie", "ID", topMovieResult.ID, "title", topMovieResult.Title)
				continue
			}

			var actorList []string
			for _, castMember := range castResult.Cast {
				actorCounts[castMember.Name] = append(actorCounts[castMember.Name], topMovieResult.Title)
				actorList = append(actorList, castMember.Name)
			}

			// Store result in cache
			cacheMutex.Lock()
			cache[slug] = FilmDetails{Title: topMovieResult.Title, Cast: actorList}
			cacheMutex.Unlock()
		}

		if w != nil {
			sendMapAsSSEData(*w, map[string]int{
				"progress": i + 1,
			})
		}
	}

	sortedActors := cleanActorMovies(actorCounts)

	saveCache()

	return sortedActors
}

type actorEntry struct {
	Name   string
	Movies []string
}

func cleanActorMovies(actorMovies map[string][]string) []actorEntry {
	// Filter out actors appearing only once and sort by movies descending
	var sortedActors []actorEntry
	for actor, movies := range actorMovies {
		if len(movies) > 1 {
			sortedActors = append(sortedActors, actorEntry{Name: actor, Movies: movies})
		}
	}

	sort.Slice(sortedActors, func(i, j int) bool {
		return len(sortedActors[i].Movies) > len(sortedActors[j].Movies)
	})

	return sortedActors
}
