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
	var wg sync.WaitGroup
	numPages := fetchNumberOfPages(username)
	results := make(chan []string, numPages)

	for i := range numPages {
		wg.Add(1)
		url := getLetterboxdURL(username, i+1)
		go func(url string, wg *sync.WaitGroup, results chan<- []string) {
			defer wg.Done()
			results <- extractFilmSlugsFromURL(url)
		}(url, &wg, results)
	}

	wg.Wait()
	close(results)

	var filmSlugs []string
	for res := range results {
		filmSlugs = append(filmSlugs, res...)
	}

	return filmSlugs
}

func fetchNumberOfPages(username string) int {
	url := getLetterboxdURL(username, 1)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching the URL:", err)
		return 1
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Println("Error loading HTML document:", err)
		return 1
	}

	lastPageElement := doc.Find("li.paginate-page").Last()

	// Convert string to int
	lastPage, err := strconv.Atoi(lastPageElement.Text())
	if err != nil {
		return 1
	}
	return lastPage
}
