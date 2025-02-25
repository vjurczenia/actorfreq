package main

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestFetchActorCountsForUser(t *testing.T) {
	// Create mock server with an HTML response
	testLetterboxdServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><div data-film-slug='saving-private-ryan' /><div data-film-slug='toy-story' /></body></html>"))
	}))
	defer testLetterboxdServer.Close()

	// Mock URL functions
	originalGetLetterboxdURL := getLetterboxdURL
	getLetterboxdURL = func(username string, page int) string { return testLetterboxdServer.URL }

	// Restore original functions after the test
	defer func() {
		getLetterboxdURL = originalGetLetterboxdURL
	}()

	cache := map[string][]string{
		"saving-private-ryan": {"Tom Hanks", "Matt Damon"},
		"toy-story":           {"Tom Hanks"},
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
