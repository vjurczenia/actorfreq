package main

import (
	"encoding/json"
	"fmt"
	"html/template"
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

	// Set the headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	actorCounts := fetchActorCounts(username, lastNMovies, &w)

	sendMapAsSSEData(w, map[string][]actorEntry{
		"actors": actorCounts,
	})

	saveCache()
}

func sendMapAsSSEData[K comparable, V any](w http.ResponseWriter, m map[K]V) {
	// Serialize the map to JSON
	md, err := json.Marshal(m)
	if err != nil {
		http.Error(w, "Failed to marshal JSON", http.StatusInternalServerError)
		return
	}

	// Send the JSON object as SSE data
	fmt.Fprintf(w, "data: %s\n\n", string(md))

	// Flush the response so the data is sent immediately
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}
