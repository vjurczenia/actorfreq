package main

import (
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestFetchMovieCredits(t *testing.T) {
	initialGetTMDBAccessToken := getTMDBAccessToken
	defer func() { getTMDBAccessToken = initialGetTMDBAccessToken }()
	getTMDBAccessToken = func() string { return "TMDB_ACCESS_TOKEN" }

	expectedHTTPCallCounts := map[string]int{
		"https://api.themoviedb.org/3/movie/1234/credits": 1,
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

	expectedMovieCredits := MovieCredits{
		Cast: []CastMember{
			{Name: "Tom Hanks"},
		},
	}
	actualMovieCredits, _ := fetchMovieCredits(1234)
	if !reflect.DeepEqual(expectedMovieCredits, *actualMovieCredits) {
		t.Errorf("Expected MovieCredits %v, got %v", expectedMovieCredits, *actualMovieCredits)
	}

	for key := range actualHTTPCallCounts {
		if expectedHTTPCallCounts[key] != actualHTTPCallCounts[key] {
			t.Errorf("Expected %d calls to %q, got %d", expectedHTTPCallCounts[key], key, actualHTTPCallCounts[key])
		}
	}
}
