package main

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

func fetchMovieSlugs(username string) []string {
	var wg sync.WaitGroup
	numPages := fetchNumberOfPages(username)
	results := make(chan []string, numPages)

	for i := range numPages {
		wg.Add(1)
		url := fmt.Sprintf("https://letterboxd.com/%s/films/by/date/page/%d", username, i+1)
		go func(url string, wg *sync.WaitGroup, results chan<- []string) {
			defer wg.Done()
			results <- extractFilmSlugsFromURL(url)
		}(url, &wg, results)
	}

	wg.Wait()
	close(results)

	var movieSlugs []string
	for res := range results {
		movieSlugs = append(movieSlugs, res...)
	}

	return movieSlugs
}

func fetchNumberOfPages(username string) int {
	url := fmt.Sprintf("https://letterboxd.com/%s/films/by/date/page/%d", username, 1)

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

	lastPage := doc.Find("li.paginate-page").Last()

	// Convert string to int
	num, err := strconv.Atoi(lastPage.Text())
	if err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	return num
}
