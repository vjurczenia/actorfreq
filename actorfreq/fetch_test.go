package actorfreq

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
				`<div data-film-slug="thor-ragnarok" />` +
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

	setUpInMemorySQLiteDB()
	cacheDB.Create(
		&Film{
			Slug:  "saving-private-ryan",
			Title: "Saving Private Ryan",
			Cast: []Credit{
				{Actor: "Tom Hanks", Roles: "Captain Miller"},
				{Actor: "Matt Damon", Roles: "Private Ryan"},
			},
		},
	)
	cacheDB.Create(
		&Film{
			Slug:  "thor-ragnarok",
			Title: "Thor: Ragnarok",
			Cast: []Credit{
				{Actor: "Matt Damon", Roles: "Actor Loki (uncredited)"},
			},
		},
	)

	rc := requestConfig{
		sortStrategy: "date",
		topNMovies:   3,
		roleFilters:  []string{"uncredited"},
	}
	actualActors := fetchActors("testUser", rc, nil)

	expectedActors := []actorDetails{
		{
			Name: "Tom Hanks",
			Movies: []movieDetails{
				{FilmSlug: "toy-story", Title: "Toy Story", Roles: "Woody"},
				{FilmSlug: "saving-private-ryan", Title: "Saving Private Ryan", Roles: "Captain Miller"},
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
