package main

import (
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

func TestFetchActorCounts(t *testing.T) {
	initialGetTMDBAccessToken := getTMDBAccessToken
	defer func() { getTMDBAccessToken = initialGetTMDBAccessToken }()
	getTMDBAccessToken = func() string { return "TMDB_ACCESS_TOKEN" }

	actualHTTPCallCounts := make(map[string]int)
	expectedHTTPCallCounts := map[string]int{
		"https://letterboxd.com/testUser/films/by/date/page/1":      1,
		"https://letterboxd.com/testUser/films/by/date/page/2":      1,
		"https://letterboxd.com/testUser/films/by/date/page/3":      1,
		"https://api.themoviedb.org/3/search/movie?query=toy-story": 1,
		"https://api.themoviedb.org/3/movie/1234/credits":           1,
	}
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
			responseString = `<div data-film-slug="saving-private-ryan" />` +
				`<ul><li class="paginate-page">3</li>` +
				`<li class="paginate-page"><a>2</a></li></ul>`
		case "https://letterboxd.com/testUser/films/by/date/page/2":
			responseString = `<div data-film-slug="toy-story" />`
		case "https://api.themoviedb.org/3/search/movie?query=toy-story":
			responseString = `{"results":[{"id":1234}]}`
		case "https://api.themoviedb.org/3/movie/1234/credits":
			responseString = `{"cast": [{"name": "Tom Hanks"}]}`
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

	cache := map[string][]string{
		"saving-private-ryan": {"Tom Hanks", "Matt Damon"},
		// "toy-story":           {"Tom Hanks"},
	}

	actualActorCounts := fetchActorCounts("testUser", cache)

	expectedActorCounts := []actorEntry{
		{Name: "Tom Hanks", Count: 2},
	}
	actorCountsAreEqual := slices.EqualFunc(expectedActorCounts, actualActorCounts, func(x actorEntry, y actorEntry) bool {
		return x.Name == y.Name && x.Count == y.Count
	})
	if !actorCountsAreEqual {
		t.Errorf("Expected actorCounts %v, got %v", expectedActorCounts, actualActorCounts)
	}

	for key := range actualHTTPCallCounts {
		if expectedHTTPCallCounts[key] != actualHTTPCallCounts[key] {
			t.Errorf("Expected %d calls to %q, got %d", expectedHTTPCallCounts[key], key, actualHTTPCallCounts[key])
		}
	}
}
