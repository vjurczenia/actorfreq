package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

const cacheFile = "cache.json" // File where cache will be saved

// loadCache loads the cached actors from a file (cache.json).
func loadCache() map[string][]string {
	cache := make(map[string][]string)

	// Check if the cache file exists
	if _, err := os.Stat(cacheFile); err == nil {
		// File exists, read it
		data, err := ioutil.ReadFile(cacheFile)
		if err != nil {
			fmt.Println("Error reading cache file:", err)
			return cache
		}

		// Unmarshal the cache data into the map
		if err := json.Unmarshal(data, &cache); err != nil {
			fmt.Println("Error unmarshalling cache data:", err)
		}
	}

	return cache
}

// saveCache persists the cache to a file (cache.json).
func saveCache(cache map[string][]string) {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling cache data:", err)
		return
	}

	if err := ioutil.WriteFile(cacheFile, data, 0644); err != nil {
		fmt.Println("Error writing cache file:", err)
	}
}
