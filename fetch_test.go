package main

import (
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
)

type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (fn RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestFetchActorCountsForUser(t *testing.T) {
	initialGetTMDBAPIKeyFunc := getTMDBAPIKey
	defer func() { getTMDBAPIKey = initialGetTMDBAPIKeyFunc }()

	getTMDBAPIKey = func() string { return "TMDB_API_KEY" }

	initialTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = initialTransport }()

	http.DefaultTransport = RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		fmt.Println(req.URL)
		var responseString string
		switch req.URL.String() {
		case "https://letterboxd.com/testUser/films/by/date/page/1":
			responseString = `<div data-film-slug="saving-private-ryan" />` +
				`<ul><li class="paginate-page">3</li>` +
				`<li class="paginate-page"><a>2</a></li></ul>`
		case "https://letterboxd.com/testUser/films/by/date/page/2":
			responseString = `<div data-film-slug="toy-story" />`
		case "https://api.themoviedb.org/3/search/movie?api_key=TMDB_API_KEY&query=toy-story":
			responseString = `{"results":[{"id":1234}]}`
		case "https://api.themoviedb.org/3/movie/1234/credits?api_key=TMDB_API_KEY":
			responseString = `{"cast": [{"name": "Tom Hanks"}]}`
		default:
			responseString = ""
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(responseString)),
			Header:     make(http.Header),
		}, nil
	})

	cache := map[string][]string{
		"saving-private-ryan": {"Tom Hanks", "Matt Damon"},
		// "toy-story":           {"Tom Hanks"},
	}

	actorCounts := fetchActorCountsForUser("testUser", cache)

	expected := []actorEntry{
		{Name: "Tom Hanks", Count: 2},
	}

	areEqual := slices.EqualFunc(actorCounts, expected, func(x actorEntry, y actorEntry) bool {
		return x.Name == y.Name && x.Count == y.Count
	})

	if !areEqual {
		t.Errorf("Expected actorCounts %q, got %q", expected, actorCounts)
	}

}
