package actorfreq

import (
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"gorm.io/gorm"
)

var letterboxdMutex sync.Mutex

func fetchLetterboxdDoc(url string) *goquery.Document {
	letterboxdMutex.Lock()
	defer letterboxdMutex.Unlock()
	return fetchDoc(url)
}

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
	return fetchLetterboxdDoc(url)
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

type Credit struct {
	gorm.Model
	Actor  string
	Roles  string
	FilmID uint `gorm:"index"`
}

type Film struct {
	gorm.Model
	Slug  string `gorm:"index"`
	Title string
	Cast  []Credit
}

func fetchFilm(slug string) Film {
	url := fmt.Sprintf("https://letterboxd.com/film/%s/", slug)
	doc := fetchLetterboxdDoc(url)

	title := doc.Find("h1.filmtitle").First().Text()
	if title == "" {
		title = slug
	}

	actors := []string{}
	roles := make(map[string][]string)
	doc.Find("a[href^='/actor/']").Each(func(i int, s *goquery.Selection) {
		actor := s.Text()
		if !slices.Contains(actors, actor) {
			actors = append(actors, actor)
		}

		role, roleExists := s.Attr("title")
		if roleExists && !slices.Contains(roles[actor], role) {
			roles[actor] = append(roles[actor], role)
		}
	})

	cast := []Credit{}
	for _, actor := range actors {
		cast = append(cast, Credit{Actor: actor, Roles: strings.Join(roles[actor], " / ")})
	}

	film := Film{Slug: slug, Title: title, Cast: cast}

	saveFilmToCache(film)

	return film
}

func fetchFollowing(username string) []string {
	url := fmt.Sprintf("https://letterboxd.com/%s/following/", username)
	doc := fetchLetterboxdDoc(url)

	following := []string{}
	doc.Find("td.table-person h3 a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			following = append(following, strings.Trim(href, "/"))
		}
	})

	return following
}
