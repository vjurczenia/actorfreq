package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

// extractFilmSlugsFromURL parses the HTML of the page and extracts the slugs.
func extractFilmSlugsFromURL(url string) []string {
	// Fetch the HTML content from the given URL
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching URL:", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: non-OK HTTP status:", resp.Status)
		return nil
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil
	}

	// Parse the HTML and extract slugs
	doc, err := html.Parse(strings.NewReader(string(bodyBytes)))
	if err != nil {
		fmt.Println("Error parsing HTML:", err)
		return nil
	}

	return extractFilmSlugs(doc)
}

// extractFilmSlugs recursively parses the HTML document to extract film slugs.
func extractFilmSlugs(n *html.Node) []string {
	var slugs []string
	if n.Type == html.ElementNode {
		for _, attr := range n.Attr {
			if attr.Key == "data-film-slug" {
				slugs = append(slugs, attr.Val)
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		slugs = append(slugs, extractFilmSlugs(c)...)
	}
	return slugs
}
