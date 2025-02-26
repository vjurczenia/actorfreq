package main

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

var getLetterboxdURL = func(username string, page int) string {
	return fmt.Sprintf("https://letterboxd.com/%s/films/by/date/page/%d", username, page)
}

func fetchFilmSlugs(username string) []string {
	// Fetch film slugs and page count from first page
	doc := fetchFilmsPageDoc(username, 1)
	filmSlugs := extractFilmSlugs(doc)
	lastPageElement := doc.Find("li.paginate-page").Last()
	numPages, err := strconv.Atoi(lastPageElement.Text())
	if err != nil {
		numPages = 1
	}

	// Fetch film slugs from remaining pages in parallel
	var wg sync.WaitGroup
	results := make(chan []string, numPages)
	for page := 2; page <= numPages; page++ {
		wg.Add(1)
		go func(username string, page int, wg *sync.WaitGroup, results chan<- []string) {
			defer wg.Done()
			doc := fetchFilmsPageDoc(username, page)
			results <- extractFilmSlugs(doc)
		}(username, page, &wg, results)
	}

	// Verify that we didn't miss any pages sequentially
	page := numPages + 1
	for {
		doc := fetchFilmsPageDoc(username, page)
		filmSlugsOnPage := extractFilmSlugs(doc)
		if len(filmSlugsOnPage) == 0 {
			fmt.Println("No more film slugs found for page:", page)
			break
		}
		filmSlugs = append(filmSlugs, filmSlugsOnPage...)
		page++
	}

	// Wait for goroutines to finish and aggregate results
	wg.Wait()
	close(results)
	for res := range results {
		filmSlugs = append(filmSlugs, res...)
	}

	return filmSlugs
}

func fetchFilmsPageDoc(username string, page int) *goquery.Document {
	url := fmt.Sprintf("https://letterboxd.com/%s/films/by/date/page/%d", username, page)
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

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Println("Error loading HTML document:", err)
		return nil
	}

	return doc
}

func extractFilmSlugs(doc *goquery.Document) []string {
	var slugs []string
	doc.Find("[data-film-slug]").Each(func(i int, s *goquery.Selection) {
		if val, exists := s.Attr("data-film-slug"); exists {
			slugs = append(slugs, val)
		}
	})
	return slugs
}
