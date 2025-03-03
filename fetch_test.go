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
	initialGetTMDBAccessToken := getTMDBAccessToken
	defer func() { getTMDBAccessToken = initialGetTMDBAccessToken }()
	getTMDBAccessToken = func() string { return "TMDB_ACCESS_TOKEN" }

	expectedHTTPCallCounts := map[string]int{
		"https://letterboxd.com/testUser/films/by/date/page/1":      1,
		"https://letterboxd.com/testUser/films/by/date/page/2":      1,
		"https://letterboxd.com/testUser/films/by/date/page/3":      1,
		"https://api.themoviedb.org/3/search/movie?query=toy-story": 1,
		"https://api.themoviedb.org/3/movie/1234/credits":           1,
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
		case "https://api.themoviedb.org/3/search/movie?query=toy-story":
			responseString = `{"results":[{"id":1234, "title": "Toy Story"}]}`
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

	initialSaveCache := saveCache
	saveCache = func() {}
	defer func() { saveCache = initialSaveCache }()

	initialCache := cache
	cache = map[string]FilmDetails{
		"saving-private-ryan": {
			Title: "Saving Private Ryan",
			Cast:  []string{"Tom Hanks", "Matt Damon"},
		},
	}
	defer func() { cache = initialCache }()

	actualActors := fetchActors("testUser", "date", 2, nil)

	expectedActors := []actorDetails{
		{
			Name: "Tom Hanks",
			Movies: []movieDetails{
				{FilmSlug: "toy-story", Title: "Toy Story"},
				{FilmSlug: "saving-private-ryan", Title: "Saving Private Ryan"},
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

// These errors could cause SIGSEGV if not handled
func TestFetchActors_ErrorSearchingMovies(t *testing.T) {
	initialGetTMDBAccessToken := getTMDBAccessToken
	defer func() { getTMDBAccessToken = initialGetTMDBAccessToken }()
	getTMDBAccessToken = func() string { return "TMDB_ACCESS_TOKEN" }

	expectedHTTPCallCounts := map[string]int{
		"https://letterboxd.com/testUser/films/by/date/page/1":      1,
		"https://letterboxd.com/testUser/films/by/date/page/2":      1,
		"https://api.themoviedb.org/3/search/movie?query=toy-story": 1,
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
			responseString = `<div data-film-slug="toy-story" />`
		case "https://letterboxd.com/testUser/films/by/date/page/2":
			responseString = ""
		case "https://api.themoviedb.org/3/search/movie?query=toy-story":
			responseString = ""
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

	initialSaveCache := saveCache
	saveCache = func() {}
	defer func() { saveCache = initialSaveCache }()

	initialCache := cache
	cache = map[string]FilmDetails{}
	defer func() { cache = initialCache }()

	actualActors := fetchActors("testUser", "date", 1, nil)

	expectedActors := []actorDetails{}
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

func TestFetchActors_ErrorFetchingMovieCredits(t *testing.T) {
	initialGetTMDBAccessToken := getTMDBAccessToken
	defer func() { getTMDBAccessToken = initialGetTMDBAccessToken }()
	getTMDBAccessToken = func() string { return "TMDB_ACCESS_TOKEN" }

	expectedHTTPCallCounts := map[string]int{
		"https://letterboxd.com/testUser/films/by/date/page/1":      1,
		"https://letterboxd.com/testUser/films/by/date/page/2":      1,
		"https://api.themoviedb.org/3/search/movie?query=toy-story": 1,
		"https://api.themoviedb.org/3/movie/1234/credits":           1,
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
			responseString = `<div data-film-slug="toy-story" />`
		case "https://letterboxd.com/testUser/films/by/date/page/2":
			responseString = ""
		case "https://api.themoviedb.org/3/search/movie?query=toy-story":
			responseString = `{"results":[{"id":1234, "title": "Toy Story"}]}`
		case "https://api.themoviedb.org/3/movie/1234/credits":
			responseString = ""
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

	initialSaveCache := saveCache
	saveCache = func() {}
	defer func() { saveCache = initialSaveCache }()

	initialCache := cache
	cache = map[string]FilmDetails{}
	defer func() { cache = initialCache }()

	actualActors := fetchActors("testUser", "date", 1, nil)

	expectedActors := []actorDetails{}
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
