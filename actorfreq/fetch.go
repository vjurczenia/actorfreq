package actorfreq

import (
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

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
					FilmSlug: slug,
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

func getFilm(slug string) FilmDetails {
	var films []FilmDetails
	var result *gorm.DB
	if db != nil {
		result = db.Preload("Cast").Where("slug = ?", slug).Limit(1).Find(&films)
	}
	if result == nil || result.Error != nil || result.RowsAffected == 0 {
		slog.Info("Cache miss", "slug", slug)
		return fetchFilmDetails(slug)
	}
	slog.Info("Cache hit", "slug", slug)
	return films[0]
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
