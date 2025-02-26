package main

import (
	"encoding/json"
	"log/slog"
	"os"
)

const cacheFile = "cache.json" // File where cache will be saved
var cache map[string][]string

// loadCache loads the cached actors from a file (cache.json).
func loadCache() {
	cache = make(map[string][]string)

	// Check if the cache file exists
	if _, err := os.Stat(cacheFile); err == nil {
		// File exists, read it
		data, err := os.ReadFile(cacheFile)
		if err != nil {
			slog.Error("Error reading cache file", "error", err)
		}

		// Unmarshal the cache data into the map
		if err := json.Unmarshal(data, &cache); err != nil {
			slog.Error("Error unmarshalling cache data", "error", err)
		}
	}
}

// saveCache persists the cache to a file (cache.json).
var saveCache = func() {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		slog.Error("Error marshalling cache data", "error", err)
		return
	}

	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		slog.Error("Error writing cache file", "error", err)
	}
}
