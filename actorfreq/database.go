package actorfreq

import (
	"fmt"
	"log/slog"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var postgresDB *gorm.DB
var memDB *gorm.DB

func SetUpDB() {
	setUpPostgresDB()
	setUpInMemorySQLiteDB()
}

func setUpPostgresDB() {
	dbHost := os.Getenv("ACTORFREQ_DB_HOST")
	dbPort := os.Getenv("ACTORFREQ_DB_PORT")
	dbUser := os.Getenv("ACTORFREQ_DB_USER")
	dbPassword := os.Getenv("ACTORFREQ_DB_PASSWORD")
	dbName := os.Getenv("ACTORFREQ_DB_NAME")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var err error
	postgresDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		slog.Error("Failed to connect to the database:", "error", err)
	}

	migrateDB(postgresDB)
}

func setUpInMemorySQLiteDB() {
	disableInMemorySQLiteDB := os.Getenv("DISABLE_IN_MEMORY_SQLITE_DB")
	if disableInMemorySQLiteDB != "true" {
		var err error
		memDB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		if err != nil {
			slog.Error("Failed to connect to in-memory SQLite database")
		}

		migrateDB(memDB)
	}
}

func migrateDB(db *gorm.DB) {
	if db != nil {
		db.AutoMigrate(&FilmDetails{})
		db.AutoMigrate(&Credit{})
	}
}

func fetchCachedFilms(filmSlugs []string) []FilmDetails {
	cacheHits := []FilmDetails{}

	if memDB != nil {
		memDB.Preload("Cast").Where("slug IN (?)", filmSlugs).Find(&cacheHits)
	}

	cacheHitSlugs := []string{}

	if postgresDB != nil {
		for _, film := range cacheHits {
			cacheHitSlugs = append(cacheHitSlugs, film.Slug)
		}

		cacheMissSlugs := difference(filmSlugs, cacheHitSlugs)

		batchCacheHits := []FilmDetails{}
		batchSize := 500

		for i := 0; i < len(cacheMissSlugs); i += batchSize {
			end := min(i+batchSize, len(cacheMissSlugs))
			batchFilmSlugs := cacheMissSlugs[i:end]

			batchCacheHits = []FilmDetails{} // Clear previous batch results
			postgresDB.Preload("Cast").Where("slug IN (?)", batchFilmSlugs).Find(&batchCacheHits)

			cacheHits = append(cacheHits, batchCacheHits...)
		}
	}

	slog.Info("Finished fetching cached films",
		"numHits", len(cacheHits),
		"numMemHits", len(cacheHitSlugs),
		"numPostgresHits", len(cacheHits)-len(cacheHitSlugs),
	)
	return cacheHits
}

func fetchCachedFilm(filmSlug string) (FilmDetails, bool) {
	for _, db := range []*gorm.DB{memDB, postgresDB} {
		if db != nil {
			var films []FilmDetails
			result := db.Preload("Cast").Where("slug = ?", filmSlug).Limit(1).Find(&films)
			if result.Error == nil && len(films) > 0 {
				filmDetails := films[0]
				slog.Info("Film cache hit", "db", getDBName(db), "filmSlug", filmDetails.Slug)
				return filmDetails, true
			}
		}
	}

	return FilmDetails{}, false
}

func saveFilmToCache(filmDetails FilmDetails) {
	for _, db := range []*gorm.DB{memDB, postgresDB} {
		if db != nil {
			slog.Info("Saving film to cache", "db", getDBName(db), "filmSlug", filmDetails.Slug)
			db.Save(&filmDetails)
		}
	}
}

func getDBName(db *gorm.DB) string {
	switch db {
	case memDB:
		return "In-memory SQLite"
	case postgresDB:
		return "PostgreSQL"
	}
	return "Unknown DB"
}
