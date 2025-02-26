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

func fetchActors(slug string) []string {
	var url string
	var body []byte
	url = fmt.Sprintf("https://api.themoviedb.org/3/search/movie?query=%s", slug)
	body = fetchTMDBResponseBody(url)

	var result struct {
		Results []struct {
			ID int `json:"id"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		slog.Error("Error parsing TMDB JSON", "error", err)
		return nil
	}

	if len(result.Results) == 0 {
		lastHyphen := strings.LastIndex(slug, "-")
		if lastHyphen != -1 {
			name := slug[:lastHyphen]
			year := slug[lastHyphen+1:]
			url = fmt.Sprintf("https://api.themoviedb.org/3/search/movie?query=%s&year=%s", name, year)
			body = fetchTMDBResponseBody(url)

			if err := json.Unmarshal(body, &result); err != nil {
				slog.Error("Error parsing TMDB JSON", "error", err)
				return nil
			}
			if len(result.Results) == 0 {
				slog.Error("Movie not found", "slug", slug, "name", name, "year", year)
				return nil
			}
		}

	}

	movieID := result.Results[0].ID
	url = fmt.Sprintf("https://api.themoviedb.org/3/movie/%d/credits", movieID)
	body = fetchTMDBResponseBody(url)

	var castResult struct {
		Cast []struct {
			Name string `json:"name"`
		} `json:"cast"`
	}

	if err := json.Unmarshal(body, &castResult); err != nil {
		slog.Error("Error parsing TMDB JSON", "error", err)
		return nil
	}

	var actors []string
	for _, actor := range castResult.Cast {
		actors = append(actors, actor.Name)
	}

	return actors

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
