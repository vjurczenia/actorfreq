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

type actorDetails struct {
	Name   string
	Movies []string
}

func fetchActors(username string, lastNMovies int, w *http.ResponseWriter) []actorDetails {
	filmSlugs := fetchFilmSlugs(username)

	if lastNMovies > 0 && lastNMovies < len(filmSlugs) {
		filmSlugs = filmSlugs[:lastNMovies]
	}

	if w != nil {
		sendMapAsSSEData(*w, map[string]int{
			"total": len(filmSlugs),
		})
	}

	actors := make(map[string]*actorDetails)
	for i, slug := range filmSlugs {
		cacheMutex.Lock()
		cachedData, found := cache[slug]
		cacheMutex.Unlock()

		if found {
			slog.Info("Cache hit", "slug", slug)

			for _, actorName := range cachedData.Cast {
				actor, found := actors[actorName]
				if !found {
					actors[actorName] = &actorDetails{Name: actorName}
					actor = actors[actorName]
				}
				actor.Movies = append(actor.Movies, cachedData.Title)
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

			var actorNames []string
			for _, castMember := range castResult.Cast {
				actorName := castMember.Name
				actor, found := actors[actorName]
				if !found {
					actors[actorName] = &actorDetails{Name: actorName}
					actor = actors[actorName]
				}
				actor.Movies = append(actor.Movies, topMovieResult.Title)
				actorNames = append(actorNames, actorName)
			}

			// Store result in cache
			cacheMutex.Lock()
			cache[slug] = FilmDetails{Title: topMovieResult.Title, Cast: actorNames}
			cacheMutex.Unlock()
		}

		if w != nil {
			sendMapAsSSEData(*w, map[string]int{
				"progress": i + 1,
			})
		}
	}

	cleanedActors := cleanActors(actors)

	saveCache()

	return cleanedActors
}

func cleanActors(actors map[string]*actorDetails) []actorDetails {
	// Filter out actors appearing only once and sort by movies descending
	var cleanedActors []actorDetails
	for _, actorDetails := range actors {
		if len(actorDetails.Movies) > 1 {
			cleanedActors = append(cleanedActors, *actorDetails)
		}
	}

	sort.Slice(cleanedActors, func(i, j int) bool {
		return len(cleanedActors[i].Movies) > len(cleanedActors[j].Movies)
	})

	return cleanedActors
}
