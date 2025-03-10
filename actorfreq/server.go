package actorfreq

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"sync"
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

type requestConfig struct {
	sortStrategy string
	topNMovies   int
}

// fetchActorsHandler processes the form submission, fetches actor details, and returns JSON
func fetchActorsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	requestConfig := getRequestConfig(r)

	// Set the headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	actors := FetchActors(username, requestConfig, &w)

	sendMapAsSSEData(w, map[string][]actorDetails{
		"actors": actors,
	})
}

func getRequestConfig(r *http.Request) requestConfig {
	sortStrategy := r.FormValue("sortStrategy")
	if sortStrategy == "" {
		sortStrategy = "date"
	}

	topNMoviesFormValue := r.FormValue("topNMovies")
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
