package actorfreq

import (
	"log/slog"
	"net/http"

	"gorm.io/gorm"
)

func fetchActorsSequentially(username string, rc requestConfig, w *http.ResponseWriter) []actorDetails {
	filmSlugs := fetchFilmSlugs(username, rc.sortStrategy)

	if rc.topNMovies > 0 && rc.topNMovies < len(filmSlugs) {
		filmSlugs = filmSlugs[:rc.topNMovies]
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
			filteredRoles := filterRoles(credit.Roles, rc.roleFilters)
			if filteredRoles != "" {
				if !found {
					actors[credit.Actor] = &actorDetails{Name: credit.Actor}
					actor = actors[credit.Actor]
				}
				actor.Movies = append(actor.Movies, movieDetails{
					FilmSlug: film.Slug,
					Title:    film.Title,
					Roles:    credit.Roles,
				})
			}
		}

		if w != nil {
			sendMapAsSSEData(*w, map[string]int{
				"progress": i + 1,
			})
		}
	}

	cleanedActors := cleanActors(actors)

	return cleanedActors
}

func getFilm(slug string) Film {
	var films []Film
	var result *gorm.DB
	if cacheDB != nil {
		result = cacheDB.Preload("Cast").Where("slug = ?", slug).Limit(1).Find(&films)
	}
	if result == nil || result.Error != nil || result.RowsAffected == 0 {
		slog.Info("Sequential cache miss", "slug", slug)
		return fetchFilm(slug)
	}
	slog.Info("Sequential cache hit", "slug", slug)
	return films[0]
}
