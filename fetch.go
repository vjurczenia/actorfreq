package main

import (
	"net/http"
	"sort"
)

type actorDetails struct {
	Name   string
	Movies []movieDetails
}

type movieDetails struct {
	FilmSlug string
	Title    string
	Roles    []string
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

	actors := make(map[string]*actorDetails)
	for i, slug := range filmSlugs {
		film := getFilm(slug)
		for _, credit := range film.Cast {
			actor, found := actors[credit.Actor]
			if !found {
				actors[credit.Actor] = &actorDetails{Name: credit.Actor}
				actor = actors[credit.Actor]
			}
			actor.Movies = append(actor.Movies, movieDetails{
				FilmSlug: slug,
				Title:    film.Title,
				Roles:    credit.Roles,
			})
		}

		if w != nil {
			sendMapAsSSEData(*w, map[string]int{
				"progress": i + 1,
			})
		}
	}

	return cleanActors(actors)
}

var getFilm = func(slug string) FilmDetails {
	var films []FilmDetails
	result := db.Preload("Cast").Where("slug = ?", slug).Limit(1).Find(&films)
	if result.Error != nil || result.RowsAffected == 0 {
		return fetchFilmDetails(slug)
	}
	return films[0]
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
