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
	db.AutoMigrate(&FilmDetails{})
	db.AutoMigrate(&Credit{})
}
