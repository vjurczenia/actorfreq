package actorfreq

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var FetchActorsPath string = "fetch-actors/"

func StartServer() {
	AddHandlers("/")

	port := "8080"
	fmt.Println("Starting server on port", port)
	http.ListenAndServe(":"+port, nil)
}

func AddHandlers(root string) {
	http.HandleFunc(root, homeHandler)
	http.HandleFunc(fmt.Sprintf("%s%s", root, FetchActorsPath), fetchActorsHandler)
}

//go:embed templates
var templates embed.FS

// homeHandler renders the homepage with a form
func homeHandler(w http.ResponseWriter, r *http.Request) {
	// Using embed.FS with template.ParseFS: https://www.reddit.com/r/golang/comments/1fllizl/comment/lo69j1e/
	tmpl, err := template.ParseFS(templates, "templates/index.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "index.html", struct{ FetchActorsPath string }{FetchActorsPath: FetchActorsPath})
}

var activeRequests int32

type requestConfig struct {
	sortStrategy string
	topNMovies   int
	roleFilters  []string
}

var precacheFollowingStarted = atomic.Bool{}

// fetchActorsHandler processes the form submission, fetches actor details, and returns JSON
func fetchActorsHandler(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt32(&activeRequests, 1)
	defer atomic.AddInt32(&activeRequests, -1)

	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	username := r.Form.Get("username")
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	requestConfig := getRequestConfig(r)

	// Set the headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	requestCache.evict() // clear out expired cache items
	requestCacheKey := r.Form.Encode()
	var actors []actorDetails
	if value, found := requestCache.get(requestCacheKey); found {
		slog.Info("Request cache hit",
			"requestCacheKey", requestCacheKey,
			"numItems", len(requestCache.items),
			"totalSize", requestCache.totalSize,
		)
		actors = value
	} else {
		if os.Getenv("FETCH_ACTORS_SEQUENTIALLY") == "true" {
			actors = fetchActorsSequentially(username, requestConfig, &w)
		} else {
			actors = fetchActors(username, requestConfig, &w)
		}
		requestCache.set(requestCacheKey, actors, 10*time.Minute)
		slog.Info("Request cache updated", "numItems", len(requestCache.items), "totalSize", requestCache.totalSize)
	}

	sendMapAsSSEData(w, map[string][]actorDetails{
		"actors": actors,
	})

	followedUsersToPrecacheForMutex.Lock()
	for _, followedUser := range fetchFollowing(username) {
		if !slices.Contains(followedUsersToPrecacheFor, followedUser) {
			followedUsersToPrecacheFor = append(followedUsersToPrecacheFor, followedUser)
		}
	}
	followedUsersToPrecacheForMutex.Unlock()

	if !precacheFollowingStarted.Load() && os.Getenv("DISABLE_PRECACHE_FOLLOWING") != "true" {
		go precacheFollowing()
	}
}

func getRequestConfig(r *http.Request) requestConfig {
	sortStrategy := r.Form.Get("sortStrategy")
	if sortStrategy == "" {
		sortStrategy = "date"
	}

	topNMoviesFormValue := r.Form.Get("topNMovies")
	topNMovies := -1
	if topNMoviesFormValue != "" {
		topNMoviesInt, err := strconv.Atoi(topNMoviesFormValue)
		if err == nil {
			topNMovies = topNMoviesInt
		}
	}

	return requestConfig{
		sortStrategy: sortStrategy,
		topNMovies:   topNMovies,
		roleFilters:  r.Form["roleFilter"],
	}
}

var sseMutex sync.Mutex

func sendMapAsSSEData[K comparable, V any](w http.ResponseWriter, m map[K]V) {
	sseMutex.Lock()
	defer sseMutex.Unlock()

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
