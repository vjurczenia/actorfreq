package main

import (
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"strconv"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

func fetchFilmSlugs(username string, sortStrategy string) []string {
	// Fetch film slugs and page count from first page
	doc := fetchFilmsPageDoc(username, sortStrategy, 1)
	filmSlugsOnPage := extractFilmSlugs(doc)
	filmSlugsByPage := map[int][]string{
		1: filmSlugsOnPage,
	}
	lastPageElement := doc.Find("li.paginate-page").Last()
	numPages, err := strconv.Atoi(lastPageElement.Text())
	if err != nil {
		numPages = 1
	}

	// Fetch film slugs from remaining pages in parallel
	var wg sync.WaitGroup
	var mu sync.Mutex
	for page := 2; page <= numPages; page++ {
		wg.Add(1)
		go func(page int) {
			defer wg.Done()
			doc := fetchFilmsPageDoc(username, sortStrategy, page)
			filmSlugsOnPage := extractFilmSlugs(doc)
			mu.Lock()
			filmSlugsByPage[page] = filmSlugsOnPage
			mu.Unlock()
		}(page)
	}

	// Verify that we didn't miss any pages sequentially
	for page := numPages + 1; true; page++ {
		doc := fetchFilmsPageDoc(username, sortStrategy, page)
		filmSlugsOnPage := extractFilmSlugs(doc)
		if len(filmSlugsOnPage) == 0 {
			slog.Info("No more film slugs found", "username", username, "page", page)
			break
		}
		mu.Lock()
		filmSlugsByPage[page] = filmSlugsOnPage
		mu.Unlock()
	}

	// Wait for goroutines to finish and aggregate results
	wg.Wait()

	var pages []int
	for page := range filmSlugsByPage {
		pages = append(pages, page)
	}
	sort.Ints(pages)

	var filmSlugs []string
	for _, page := range pages {
		filmSlugs = append(filmSlugs, filmSlugsByPage[page]...)
	}

	return filmSlugs
}

func fetchFilmsPageDoc(username string, sortStrategy string, page int) *goquery.Document {
	url := fmt.Sprintf("https://letterboxd.com/%s/films/by/%s/page/%d", username, sortStrategy, page)
	return fetchDoc(url)
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

func fetchFilmDetails(slug string) FilmDetails {
	url := fmt.Sprintf("https://letterboxd.com/film/%s/", slug)
	doc := fetchDoc(url)

	title := doc.Find("h1.filmtitle").First().Text()
	if title == "" {
		title = slug
	}

	cast := []string{}
	doc.Find("a[href^='/actor/']").Each(func(i int, s *goquery.Selection) {
		actor := s.Text()
		if !slices.Contains(cast, actor) {
			cast = append(cast, actor)
		}
	})

	filmDetails := FilmDetails{Title: title, Cast: cast}

	// Store result in cache
	cacheMutex.Lock()
	cache[slug] = filmDetails
	cacheMutex.Unlock()

	return filmDetails
}
