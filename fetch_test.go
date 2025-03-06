package main

import (
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestFetchActors(t *testing.T) {
	expectedHTTPCallCounts := map[string]int{
		"https://letterboxd.com/testUser/films/by/date/page/1": 1,
		"https://letterboxd.com/testUser/films/by/date/page/2": 1,
		"https://letterboxd.com/testUser/films/by/date/page/3": 1,
		"https://letterboxd.com/film/toy-story/":               1,
	}
	actualHTTPCallCounts := make(map[string]int)
	for key := range expectedHTTPCallCounts {
		actualHTTPCallCounts[key] = 0
	}

	initialTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = initialTransport }()
	http.DefaultTransport = RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		urlString := req.URL.String()
		var responseString string
		switch urlString {
		case "https://letterboxd.com/testUser/films/by/date/page/1":
			responseString = `<div data-film-slug="toy-story" />` +
				`<ul><li class="paginate-page">3</li>` +
				`<li class="paginate-page"><a>2</a></li></ul>`
		case "https://letterboxd.com/testUser/films/by/date/page/2":
			responseString = `<div data-film-slug="saving-private-ryan" />` +
				`<div data-film-slug="forrest-gump" />`
		case "https://letterboxd.com/testUser/films/by/date/page/3":
			responseString = ""
		case "https://letterboxd.com/film/toy-story/":
			responseString = `<h1 class="filmtitle">Toy Story</h1>` +
				`<a href="/actor/tom-hanks" title="Woody">Tom Hanks</a>`
		default:
			responseString = ""
		}
		actualHTTPCallCounts[urlString]++
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(responseString)),
			Header:     make(http.Header),
		}, nil
	})

	initialCacheResult := cacheResult
	cacheResult = func(fd FilmDetails) {}
	defer func() { cacheResult = initialCacheResult }()

	initialGetFilm := getFilm
	getFilm = func(slug string) FilmDetails {
		uncached := []string{"toy-story"}

		if slices.Contains(uncached, slug) {
			return fetchFilmDetails(slug)
		}

		cache := map[string]FilmDetails{
			"saving-private-ryan": {
				Title: "Saving Private Ryan",
				Cast: []Credit{
					{Actor: "Tom Hanks", Roles: []string{"Captain Miller"}},
					{Actor: "Matt Damon", Roles: []string{"Private Ryan"}},
				},
			},
		}

		cachedData, found := cache[slug]
		if !found {
			t.Errorf("Expected slug %s not found in cache", slug)
		}

		return cachedData
	}
	defer func() { getFilm = initialGetFilm }()

	actualActors := fetchActors("testUser", "date", 2, nil)

	expectedActors := []actorDetails{
		{
			Name: "Tom Hanks",
			Movies: []movieDetails{
				{FilmSlug: "toy-story", Title: "Toy Story", Roles: []string{"Woody"}},
				{FilmSlug: "saving-private-ryan", Title: "Saving Private Ryan", Roles: []string{"Captain Miller"}},
			},
		},
	}
	actorsAreEqual := slices.EqualFunc(expectedActors, actualActors, func(x actorDetails, y actorDetails) bool {
		return x.Name == y.Name && reflect.DeepEqual(x.Movies, y.Movies)
	})
	if !actorsAreEqual {
		t.Errorf("Expected actors %v, got %v", expectedActors, actualActors)
	}

	for key := range actualHTTPCallCounts {
		if expectedHTTPCallCounts[key] != actualHTTPCallCounts[key] {
			t.Errorf("Expected %d calls to %q, got %d", expectedHTTPCallCounts[key], key, actualHTTPCallCounts[key])
		}
	}
}
