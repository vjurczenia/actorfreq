package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sync"
)

var (
	cacheLock sync.Mutex
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
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	cacheLock.Lock()
	cache := loadCache()
	cacheLock.Unlock()

	actorCounts := fetchActorCounts(username, cache)

	cacheLock.Lock()
	saveCache(cache)
	cacheLock.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(actorCounts)
}
