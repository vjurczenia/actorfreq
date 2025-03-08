package actorfreq

import (
	"net/http"
	"sort"

	"gorm.io/gorm"
)

type actorDetails struct {
	Name   string
	Movies []movieDetails
}

type movieDetails struct {
	FilmSlug string
	Title    string
	Roles    string
}

func FetchActors(username string, sortStrategy string, topNMovies int, w *http.ResponseWriter) []actorDetails {
	filmSlugs := fetchFilmSlugs(username, sortStrategy)

	if topNMovies > 0 && topNMovies < len(filmSlugs) {
		filmSlugs = filmSlugs[:topNMovies]
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

func getFilm(slug string) FilmDetails {
	var films []FilmDetails
	var result *gorm.DB
	if db != nil {
		result = db.Preload("Cast").Where("slug = ?", slug).Limit(1).Find(&films)
	}
	if result != nil && (result.Error != nil || result.RowsAffected == 0) {
		return fetchFilmDetails(slug)
	}
	return films[0]
}

func cleanActors(actors map[string]*actorDetails) []actorDetails {
	// Filter out actors appearing only once and sort by movies descending and name ascending
	cleanedActors := []actorDetails{}
	for _, actor := range actors {
		if len(actor.Movies) > 1 {
			cleanedActors = append(cleanedActors, *actor)
		}
	}

	sort.Slice(cleanedActors, func(i, j int) bool {
		if len(cleanedActors[i].Movies) == len(cleanedActors[j].Movies) {
			return cleanedActors[i].Name < cleanedActors[j].Name
		}
		return len(cleanedActors[i].Movies) > len(cleanedActors[j].Movies)
	})

	return cleanedActors
}
