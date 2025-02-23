package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// https://developer.themoviedb.org/docs/rate-limiting
const requestInterval = time.Second / 50 // Limit to 50 requests per second

// fetchActorsForPage fetches actors for a specific page URL.
func fetchActorsForPage(url, tmdbAPIKey string, cache map[string][]string) []string {
	// You can make a request to the URL to parse HTML and extract film slugs
	slugs := extractFilmSlugsFromURL(url)
	var actors []string
	for _, slug := range slugs {
		// Check if actors are cached
		if cachedActors, found := cache[slug]; found {
			actors = append(actors, cachedActors...)
			fmt.Println("Cache hit for slug:", slug)
		} else {
			actors = fetchActorsFromTMDB(slug, tmdbAPIKey)
			cache[slug] = actors // Cache the result
		}
		time.Sleep(requestInterval) // Rate limit API calls
	}
	return actors
}

// fetchActorsFromTMDB fetches the actors for a given movie slug from TMDB API.
func fetchActorsFromTMDB(slug string, tmdbAPIKey string) []string {
	apiURL := fmt.Sprintf("https://api.themoviedb.org/3/search/movie?api_key=%s&query=%s", tmdbAPIKey, slug)
	resp, err := http.Get(apiURL)
	if err != nil {
		fmt.Println("Error fetching TMDB data:", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: non-OK TMDB API status:", resp.Status)
		return nil
	}

	var result struct {
		Results []struct {
			ID int `json:"id"`
		} `json:"results"`
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading TMDB response:", err)
		return nil
	}

	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Println("Error parsing TMDB JSON:", err)
		return nil
	}

	if len(result.Results) == 0 {
		fmt.Println("No movie found for slug:", slug)
		return nil
	}

	movieID := result.Results[0].ID
	return fetchMovieCast(movieID, tmdbAPIKey)
}

// fetchMovieCast fetches the cast for a specific movie by ID.
func fetchMovieCast(movieID int, tmdbAPIKey string) []string {
	apiURL := fmt.Sprintf("https://api.themoviedb.org/3/movie/%d/credits?api_key=%s", movieID, tmdbAPIKey)
	resp, err := http.Get(apiURL)
	if err != nil {
		fmt.Println("Error fetching movie cast:", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: non-OK TMDB API status:", resp.Status)
		return nil
	}

	var castResult struct {
		Cast []struct {
			Name string `json:"name"`
		} `json:"cast"`
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading TMDB cast response:", err)
		return nil
	}

	if err := json.Unmarshal(body, &castResult); err != nil {
		fmt.Println("Error parsing TMDB cast JSON:", err)
		return nil
	}

	var actors []string
	for _, actor := range castResult.Cast {
		actors = append(actors, actor.Name)
	}

	return actors
}
