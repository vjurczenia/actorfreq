package actorfreq

import (
	"log/slog"
	"net/http"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (fn RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func setupTestDB() {
	var err error
	db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{}) // In-memory DB for testing
	if err != nil {
		slog.Error("Failed to connect to test database")
	}

	migrateDB()
}
