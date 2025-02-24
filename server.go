package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"sync"
)

var (
	cacheLock sync.Mutex
)

func startServer() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/fetch", fetchHandler)

	port := "8080"
	fmt.Println("Starting server on port", port)
	http.ListenAndServe(":"+port, nil)
}

// homeHandler renders the homepage with a form
func homeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// fetchHandler processes the form submission, fetches actors, and returns JSON
func fetchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	// Load the API key from environment variables
	tmdbAPIKey := os.Getenv("TMDB_API_KEY")
	if tmdbAPIKey == "" {
		http.Error(w, "Missing TMDB API key", http.StatusInternalServerError)
		return
	}

	// Fetch actors (using caching)
	cacheLock.Lock()
	cache := loadCache()
	cacheLock.Unlock()

	actorCounts := fetchActorsForUser(username, tmdbAPIKey, cache)

	// Save the updated cache
	cacheLock.Lock()
	saveCache(cache)
	cacheLock.Unlock()

	sortedActors := sortActorCounts(actorCounts)

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sortedActors)
}
