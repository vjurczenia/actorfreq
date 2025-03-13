package actorfreq

import (
	"fmt"
	"log/slog"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func SetUpDB() {
	dbHost := os.Getenv("ACTORFREQ_DB_HOST")
	dbPort := os.Getenv("ACTORFREQ_DB_PORT")
	dbUser := os.Getenv("ACTORFREQ_DB_USER")
	dbPassword := os.Getenv("ACTORFREQ_DB_PASSWORD")
	dbName := os.Getenv("ACTORFREQ_DB_NAME")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		slog.Error("Failed to connect to the database:", "error", err)
	}

	migrateDB()
}

func migrateDB() {
	if db != nil {
		db.AutoMigrate(&FilmDetails{})
		db.AutoMigrate(&Credit{})
	}
}

func fetchCachedFilms(filmSlugs []string) []FilmDetails {
	cacheHits := []FilmDetails{}

	if db != nil {
		batchCacheHits := []FilmDetails{}
		batchSize := 500

		for i := 0; i < len(filmSlugs); i += batchSize {
			end := min(i+batchSize, len(filmSlugs))
			batchFilmSlugs := filmSlugs[i:end]

			batchCacheHits = []FilmDetails{} // Clear previous batch results
			db.Preload("Cast").Where("slug IN (?)", batchFilmSlugs).Find(&batchCacheHits)

			cacheHits = append(cacheHits, batchCacheHits...)
		}
	}

	return cacheHits
}

func fetchCachedFilm(filmSlug string) (FilmDetails, bool) {
	if db != nil {
		var films []FilmDetails
		result := db.Preload("Cast").Where("slug = ?", filmSlug).Limit(1).Find(&films)
		if result.Error == nil && len(films) > 0 {
			return films[0], true
		}
	}
	return FilmDetails{}, false
}

func storeFilmToCache(filmDetails FilmDetails) {
	if db != nil {
		db.Create(&filmDetails)
	}
}
