package actorfreq

import (
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
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

	films := getFilms(filmSlugs, w)
	actors := make(map[string]*actorDetails)
	for _, film := range films {
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
	}

	cleanedActors := cleanActors(actors)

	return cleanedActors
}

func getFilms(filmSlugs []string, w *http.ResponseWriter) []Film {
	cacheHits := fetchCachedFilms(filmSlugs)

	progress := len(cacheHits)
	if w != nil {
		sendMapAsSSEData(*w, map[string]int{
			"progress": progress,
		})
	}

	filmsMap := make(map[string]Film)
	for _, film := range cacheHits {
		filmsMap[film.Slug] = film
	}

	for _, filmSlug := range filmSlugs {
		_, exists := filmsMap[filmSlug]
		if !exists {
			filmsMap[filmSlug] = fetchFilm(filmSlug)
			if w != nil {
				progress++
				sendMapAsSSEData(*w, map[string]int{
					"progress": progress,
				})
			}
		}
	}

	var films []Film
	for _, filmSlug := range filmSlugs {
		films = append(films, filmsMap[filmSlug])
	}

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
	if postgresDB != nil || memDB != nil {
		defer precacheFollowingStarted.Store(false)
		for {
			if atomic.LoadInt32(&activeRequests) == 0 {
				if len(filmSlugsToPrecache) != 0 {
					slug := filmSlugsToPrecache[0]

					_, exists := fetchCachedFilm(slug)
					if !exists {
						slog.Info("Precaching followedUser film slug", "slug", slug)
						fetchFilm(slug)
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
