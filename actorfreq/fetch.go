package actorfreq

import (
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
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

func FetchActors(username string, rc requestConfig, w *http.ResponseWriter) []actorDetails {
	filmSlugs := fetchFilmSlugs(username, rc.sortStrategy)

	if rc.topNMovies > 0 && rc.topNMovies < len(filmSlugs) {
		filmSlugs = filmSlugs[:rc.topNMovies]
	}

	if w != nil {
		sendMapAsSSEData(*w, map[string]int{
			"total": len(filmSlugs),
		})
	}

	films := getFilms(filmSlugs)
	actors := make(map[string]*actorDetails)
	for i, film := range films {
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

	return cleanActors(actors)
}

func getFilms(filmSlugs []string) []FilmDetails {
	start := time.Now()

	cacheHits := []FilmDetails{}

	if db != nil {
		batchCacheHits := []FilmDetails{}
		batchSize := 500

		for i := 0; i < len(filmSlugs); i += batchSize {
			end := min(i+batchSize, len(filmSlugs))
			batchFilmSlugs := filmSlugs[i:end]

			batchCacheHits = []FilmDetails{} // Clear previous batch results
			db.Preload("Cast").Where("slug IN (?)", batchFilmSlugs).Find(&batchCacheHits)

			cacheHits = append(cacheHits, batchCacheHits...)
		}
	}

	filmsMap := make(map[string]FilmDetails)
	for _, film := range cacheHits {
		filmsMap[film.Slug] = film
	}

	for _, filmSlug := range filmSlugs {
		_, exists := filmsMap[filmSlug]
		if !exists {
			filmsMap[filmSlug] = fetchFilmDetails(filmSlug)
		}
	}

	var films []FilmDetails
	for _, filmSlug := range filmSlugs {
		films = append(films, filmsMap[filmSlug])
	}

	elapsed := time.Since(start)
	slog.Info("Execution time", "elapsed", elapsed)

	return films
}

func filterRoles(roles string, roleFilters []string) string {
	for _, roleFilter := range roleFilters {
		switch roleFilter {
		case "additional_voices":
			if roles == "Additional Voices" || roles == "Additional Voices (voice)" {
				return ""
			}
		case "voice":
			if strings.HasSuffix(roles, "(voice)") {
				return ""
			}
		case "uncredited":
			if strings.HasSuffix(roles, "(uncredited)") {
				return ""
			}
		}
	}
	return roles
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

var filmSlugsToPrecacheMutex sync.Mutex
var filmSlugsToPrecache = []string{}
var followedUsersToPrecacheForMutex sync.Mutex
var followedUsersToPrecacheFor = []string{}

func precacheFollowing() {
	precacheFollowingStarted.Store(true)
	if db != nil {
		defer precacheFollowingStarted.Store(false)
		for {
			if atomic.LoadInt32(&activeRequests) == 0 {
				if len(filmSlugsToPrecache) != 0 {
					slug := filmSlugsToPrecache[0]
					var films []FilmDetails
					result := db.Preload("Cast").Where("slug = ?", slug).Limit(1).Find(&films)
					if result.Error != nil || result.RowsAffected == 0 {
						slog.Info("Precaching followedUser film slug", "slug", slug)
						fetchFilmDetails(slug)
					}

					filmSlugsToPrecacheMutex.Lock()
					filmSlugsToPrecache = filmSlugsToPrecache[1:]
					filmSlugsToPrecacheMutex.Unlock()
				} else if len(followedUsersToPrecacheFor) != 0 {
					followedUser := followedUsersToPrecacheFor[0]
					slog.Info("Fetching followedUser film slugs", "followedUser", followedUser)

					filmSlugsToPrecacheMutex.Lock()
					filmSlugsToPrecache = fetchFilmSlugs(followedUser, "release")
					filmSlugsToPrecacheMutex.Unlock()

					followedUsersToPrecacheForMutex.Lock()
					followedUsersToPrecacheFor = followedUsersToPrecacheFor[1:]
					followedUsersToPrecacheForMutex.Unlock()
				} else {
					break
				}
			}
		}
	}
}
