package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"sync"
)

var cache = make(map[string]FilmDetails)
var cacheMutex = &sync.Mutex{}

const cacheFile = "cache.json"

func loadCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	file, err := os.Open(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		slog.Error("Error opening cache file", "error", err)
		return
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cache); err != nil {
		slog.Error("Error decoding cache file", "error", err)
	}
}

var saveCache = func() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		slog.Error("Error marshalling cache data", "error", err)
		return
	}

	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		slog.Error("Error writing cache file", "error", err)
	}
}
