package actorfreq

import (
	"log/slog"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func initDatabaseBenchmark() {
	err := godotenv.Load()
	if err != nil {
		slog.Warn("No .env file found")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	slog.SetDefault(logger)
}

func prepareDatabaseBenchmarkDB() {
	cacheDB.Migrator().DropTable(&Film{})
	cacheDB.Migrator().DropTable(&Credit{})
	setUpGORMTables()
}

func runDatabaseBenchmark(b *testing.B) {
	prepareDatabaseBenchmarkDB()

	rc := requestConfig{
		sortStrategy: "date",
		topNMovies:   100,
		roleFilters:  []string{},
	}

	// Seed cache
	fetchActors("pablo_agave", rc, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fetchActors("pablo_agave", rc, nil)
	}
}

func BenchmarkPostgresDBPerformance(b *testing.B) {
	initDatabaseBenchmark()

	os.Setenv("ACTORFREQ_DB_NAME", "actorfreq_test")
	setUpPostgresDB()

	runDatabaseBenchmark(b)
}

func BenchmarkInMemorySQLiteDBPerformance(b *testing.B) {
	initDatabaseBenchmark()

	setUpInMemorySQLiteDB()

	runDatabaseBenchmark(b)
}
