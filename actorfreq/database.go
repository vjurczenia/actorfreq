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
	disablePostgresDB := os.Getenv("DISABLE_POSTGRES_DB")
	if disablePostgresDB != "true" {
		setUpPostgresDB()
	}

	disableInMemorySQLiteDB := os.Getenv("DISABLE_IN_MEMORY_SQLITE_DB")
	if disableInMemorySQLiteDB != "true" {
		setUpInMemorySQLiteDB()
	}
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
	var err error
	memDB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		slog.Error("Failed to connect to in-memory SQLite database")
	}

	migrateDB(memDB)
}

func migrateDB(db *gorm.DB) {
	if db != nil {
		db.AutoMigrate(&Film{})
		db.AutoMigrate(&Credit{})
	}
}

func fetchCachedFilms(filmSlugs []string) []Film {
	cacheHits := []Film{}

	if memDB != nil {
		memDB.Preload("Cast").Where("slug IN (?)", filmSlugs).Find(&cacheHits)
	}

	cacheHitSlugs := []string{}

	if postgresDB != nil {
		for _, film := range cacheHits {
			cacheHitSlugs = append(cacheHitSlugs, film.Slug)
		}

		cacheMissSlugs := difference(filmSlugs, cacheHitSlugs)

		batchCacheHits := []Film{}
		batchSize := 500

		for i := 0; i < len(cacheMissSlugs); i += batchSize {
			end := min(i+batchSize, len(cacheMissSlugs))
			batchFilmSlugs := cacheMissSlugs[i:end]

			batchCacheHits = []Film{} // Clear previous batch results
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

func fetchCachedFilm(filmSlug string) (Film, bool) {
	for _, db := range []*gorm.DB{memDB, postgresDB} {
		if db != nil {
			var films []Film
			result := db.Preload("Cast").Where("slug = ?", filmSlug).Limit(1).Find(&films)
			if result.Error == nil && len(films) > 0 {
				films := films[0]
				slog.Info("Film cache hit", "db", getDBName(db), "filmSlug", films.Slug)
				return films, true
			}
		}
	}

	return Film{}, false
}

func saveFilmToCache(films Film) {
	for _, db := range []*gorm.DB{memDB, postgresDB} {
		if db != nil {
			slog.Info("Saving film to cache", "db", getDBName(db), "filmSlug", films.Slug)
			// GORM populates the ID after inserting into memDB, causing a conflict when inserting the same object into postgresDB.
			// Handle by resetting the ID before inserting.
			films.ID = 0
			db.Create(&films)
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
