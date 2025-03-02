package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

type MovieSearchResult struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

type MovieSearchResults struct {
	Results []MovieSearchResult `json:"results"`
}

func searchMovie(slug string) (*MovieSearchResults, error) {
	url := fmt.Sprintf("https://api.themoviedb.org/3/search/movie?query=%s", slug)
	body := fetchTMDBResponseBody(url)

	var result MovieSearchResults

	if err := json.Unmarshal(body, &result); err != nil {
		slog.Error("Error parsing TMDB JSON", "error", err)
		return nil, err
	}

	// Incompatible film slug?
	if len(result.Results) == 0 {
		lastHyphen := strings.LastIndex(slug, "-")
		if lastHyphen != -1 {
			name := slug[:lastHyphen]
			year := slug[lastHyphen+1:]
			url = fmt.Sprintf("https://api.themoviedb.org/3/search/movie?query=%s&year=%s", name, year)
			body = fetchTMDBResponseBody(url)

			if err := json.Unmarshal(body, &result); err != nil {
				slog.Error("Error parsing TMDB JSON", "error", err)
				return nil, err
			}
			if len(result.Results) == 0 {
				slog.Error("Movie not found", "slug", slug, "name", name, "year", year)
				return nil, fmt.Errorf("movie %s not found", slug)
			}
		}
	}

	return &result, nil
}

type CastMember struct {
	Name string `json:"name"`
}

type MovieCredits struct {
	Cast []CastMember `json:"cast"`
}

func fetchMovieCredits(movieID int) (*MovieCredits, error) {
	url := fmt.Sprintf("https://api.themoviedb.org/3/movie/%d/credits", movieID)
	body := fetchTMDBResponseBody(url)

	var result MovieCredits
	if err := json.Unmarshal(body, &result); err != nil {
		slog.Error("Error parsing TMDB JSON", "error", err)
		return nil, err
	}

	return &result, nil
}

func fetchTMDBResponseBody(url string) []byte {
	// https://developer.themoviedb.org/docs/rate-limiting
	const requestInterval = time.Second / 50 // Limit to 50 requests per second
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		slog.Error("Error creating TMDB request", "error", err)
		return nil
	}

	tmdbAccessToken := getTMDBAccessToken()
	bearerToken := fmt.Sprintf("Bearer %s", tmdbAccessToken)
	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", bearerToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("Error fetching TMDB data", "error", err)
		return nil
	}
	defer resp.Body.Close()
	time.Sleep(requestInterval) // Rate limit API calls

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Error reading TMDB response", "error", err)
		return nil
	}

	return body
}

var getTMDBAccessToken = func() string {
	tmdbAccessToken := os.Getenv("TMDB_ACCESS_TOKEN")
	if tmdbAccessToken == "" {
		slog.Error("TMDB Access Token is missing from the .env file.")
	}
	return tmdbAccessToken
}
