package actorfreq

import (
	"log/slog"
	"net/http"

	"github.com/PuerkitoBio/goquery"
)

func fetchDoc(url string) *goquery.Document {
	resp, err := http.Get(url)
	if err != nil {
		slog.Error("Error fetching URL", "error", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("Error: non-OK HTTP status", "status", resp.Status)
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		slog.Error("Error loading HTML document", "error", err)
		return nil
	}

	return doc
}

func difference(a, b []string) []string {
	setB := make(map[string]struct{})
	for _, item := range b {
		setB[item] = struct{}{}
	}

	var result []string
	for _, item := range a {
		if _, found := setB[item]; !found {
			result = append(result, item)
		}
	}

	return result
}
