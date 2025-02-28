package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
)

func startServer() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/fetch-actor-counts", fetchActorCountsHandler)

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

// fetchActorCountsHandler processes the form submission, fetches actor counts, and returns JSON
func fetchActorCountsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	lastNMoviesFormValue := r.FormValue("lastNMovies")
	lastNMovies := -1
	if lastNMoviesFormValue != "" {
		lastNMoviesInt, err := strconv.Atoi(lastNMoviesFormValue)
		if err == nil {
			lastNMovies = lastNMoviesInt
		}
	}

	// actorCounts := fetchActorCounts(username, lastNMovies)

	// w.Header().Set("Content-Type", "application/json")
	// json.NewEncoder(w).Encode(actorCounts)

	// Set the headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	filmSlugs := fetchFilmSlugs(username)

	if lastNMovies > 0 && lastNMovies < len(filmSlugs) {
		filmSlugs = filmSlugs[:lastNMovies]
	}

	progress := map[string]int{
		"numerator":   0,
		"denominator": len(filmSlugs),
	}

	// Serialize the Fraction object to JSON
	progressData, err := json.Marshal(progress)
	if err != nil {
		http.Error(w, "Failed to marshal JSON", http.StatusInternalServerError)
		return
	}

	// Send the JSON object as SSE data
	fmt.Fprintf(w, "data: %s\n\n", string(progressData))

	// Flush the response so the data is sent immediately
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	actorCounts := make(map[string]int)
	var actors []string
	for _, slug := range filmSlugs {
		if cachedActors, found := cache[slug]; found {
			slog.Info("Cache hit", "slug", slug)
			actors = cachedActors
		} else {
			slog.Info("Cache miss", "slug", slug)
			actors = fetchActors(slug)
			if actors != nil {
				cache[slug] = actors
			}
		}

		for _, actor := range actors {
			actorCounts[actor]++
		}

		progress["numerator"]++

		data, err := json.Marshal(progress)
		if err != nil {
			http.Error(w, "Failed to marshal JSON", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "data: %s\n\n", string(data))

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}

	// Filter out actors appearing only once
	for actor, count := range actorCounts {
		if count < 2 {
			delete(actorCounts, actor)
		}
	}

	sortedActors := sortActorCounts(actorCounts)

	result := map[string][]actorEntry{
		"actors": sortedActors,
	}

	// Serialize the Fraction object to JSON
	resultData, err := json.Marshal(result)
	if err != nil {
		http.Error(w, "Failed to marshal JSON", http.StatusInternalServerError)
		return
	}

	// Send the JSON object as SSE data
	fmt.Fprintf(w, "data: %s\n\n", string(resultData))

	// Flush the response so the data is sent immediately
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	saveCache()
}
