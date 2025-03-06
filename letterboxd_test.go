package main

import (
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestFetchFilmSlugs(t *testing.T) {
	actualHTTPCallCounts := make(map[string]int)
	expectedHTTPCallCounts := map[string]int{
		"https://letterboxd.com/testUser/films/by/date/page/1": 1,
		"https://letterboxd.com/testUser/films/by/date/page/2": 1,
		"https://letterboxd.com/testUser/films/by/date/page/3": 1,
		"https://letterboxd.com/testUser/films/by/date/page/4": 1,
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
				`<ul><li class="paginate-page">3</li></ul>`
		case "https://letterboxd.com/testUser/films/by/date/page/2":
			responseString = `<div data-film-slug="forrest-gump" />`
			// Ensure the goroutine handling this request finishes after the next one
			time.Sleep(1 * time.Second)
		case "https://letterboxd.com/testUser/films/by/date/page/3":
			responseString = `<div data-film-slug="toy-story" />`
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

	actualFilmSlugs := fetchFilmSlugs("testUser", "date")

	expectedFilmSlugs := []string{
		"saving-private-ryan", "forrest-gump", "toy-story",
	}

	if !reflect.DeepEqual(expectedFilmSlugs, actualFilmSlugs) {
		t.Errorf("Expected filmSlugs %v, got %v", expectedFilmSlugs, actualFilmSlugs)
	}

	for key := range actualHTTPCallCounts {
		if expectedHTTPCallCounts[key] != actualHTTPCallCounts[key] {
			t.Errorf("Expected %d calls to %q, got %d", expectedHTTPCallCounts[key], key, actualHTTPCallCounts[key])
		}
	}
}

func TestFetchFilmDetails(t *testing.T) {
	actualHTTPCallCounts := make(map[string]int)
	expectedHTTPCallCounts := map[string]int{
		"https://letterboxd.com/film/toy-story/": 1,
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
		case "https://letterboxd.com/film/toy-story/":
			responseString = `<h1 class="filmtitle">Toy Story</h1>` +
				`<h1 class="filmtitle">NOT TOY STORY</h1>` +
				`<a href="/actor/tom-hanks" title="Woody">Tom Hanks</a>` +
				`<a href="/actor/tom-hanks" title="Another Role">Tom Hanks</a>`
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

	setupTestDB()

	actualFilmDetails := fetchFilmDetails("toy-story")

	expectedFilmDetails := FilmDetails{
		Slug:  "toy-story",
		Title: "Toy Story",
		Cast:  []Credit{{Actor: "Tom Hanks", Roles: "Woody / Another Role"}},
	}

	filmDetailsEqual := expectedFilmDetails.Slug == actualFilmDetails.Slug &&
		expectedFilmDetails.Title == actualFilmDetails.Title &&
		slices.EqualFunc(expectedFilmDetails.Cast, actualFilmDetails.Cast, func(a, b Credit) bool {
			return a.Actor == b.Actor && a.Roles == b.Roles
		})

	if !filmDetailsEqual {
		t.Errorf("Expected filmSlugs %v, got %v", expectedFilmDetails, actualFilmDetails)
	}

	for key := range actualHTTPCallCounts {
		if expectedHTTPCallCounts[key] != actualHTTPCallCounts[key] {
			t.Errorf("Expected %d calls to %q, got %d", expectedHTTPCallCounts[key], key, actualHTTPCallCounts[key])
		}
	}
}

func TestFetchFilmDetails_NoValuesOnPage(t *testing.T) {
	actualHTTPCallCounts := make(map[string]int)
	expectedHTTPCallCounts := map[string]int{
		"https://letterboxd.com/film/toy-story/": 1,
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
		case "https://letterboxd.com/film/toy-story/":
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

	setupTestDB()

	actualFilmDetails := fetchFilmDetails("toy-story")

	expectedFilmDetails := FilmDetails{
		Slug:  "toy-story",
		Title: "toy-story",
		Cast:  []Credit{},
	}

	filmDetailsEqual := expectedFilmDetails.Slug == actualFilmDetails.Slug &&
		expectedFilmDetails.Title == actualFilmDetails.Title &&
		reflect.DeepEqual(expectedFilmDetails.Cast, actualFilmDetails.Cast)

	if !filmDetailsEqual {
		t.Errorf("Expected filmSlugs %v, got %v", expectedFilmDetails, actualFilmDetails)
	}

	for key := range actualHTTPCallCounts {
		if expectedHTTPCallCounts[key] != actualHTTPCallCounts[key] {
			t.Errorf("Expected %d calls to %q, got %d", expectedHTTPCallCounts[key], key, actualHTTPCallCounts[key])
		}
	}
}
