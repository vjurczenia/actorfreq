package actorfreq

import (
	"fmt"
	"log/slog"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var cacheDB *gorm.DB

func SetUpDB() {
	if os.Getenv("DISABLE_POSTGRES_DB") != "true" {
		setUpPostgresDB()
	} else if os.Getenv("DISABLE_IN_MEMORY_SQLITE_DB") != "true" {
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
	cacheDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		slog.Error("Failed to connect to the database:", "error", err)
	}

	migrateDB(cacheDB)
}

func setUpInMemorySQLiteDB() {
	var err error
	cacheDB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		slog.Error("Failed to connect to in-memory SQLite database")
	}

	migrateDB(cacheDB)
}

func migrateDB(db *gorm.DB) {
	if db != nil {
		db.AutoMigrate(&Film{})
		db.AutoMigrate(&Credit{})
	}
}

func fetchCachedFilms(filmSlugs []string) []Film {
	cacheHits := []Film{}

	if cacheDB != nil {
		batchCacheHits := []Film{}
		batchSize := 500

		for i := 0; i < len(filmSlugs); i += batchSize {
			end := min(i+batchSize, len(filmSlugs))
			batchFilmSlugs := filmSlugs[i:end]

			batchCacheHits = []Film{} // Clear previous batch results
			cacheDB.Preload("Cast").Where("slug IN (?)", batchFilmSlugs).Find(&batchCacheHits)

			cacheHits = append(cacheHits, batchCacheHits...)
		}

		slog.Info("Finished fetching cached films", "numHits", len(cacheHits))
	}

	return cacheHits
}

func fetchCachedFilm(filmSlug string) (Film, bool) {
	if cacheDB != nil {
		var films []Film
		result := cacheDB.Preload("Cast").Where("slug = ?", filmSlug).Limit(1).Find(&films)
		if result.Error == nil && len(films) > 0 {
			films := films[0]
			slog.Info("Film cache hit", "filmSlug", films.Slug)
			return films, true
		}
	}

	return Film{}, false
}

func saveFilmToCache(films Film) {
	if cacheDB != nil {
		slog.Info("Saving film to cache", "filmSlug", films.Slug)
		cacheDB.Create(&films)
	}
}
