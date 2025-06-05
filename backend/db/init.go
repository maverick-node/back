package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Initdb() {
	dbURL := "postgresql://postgres:UMBrrHSVgkroxZUGNXTZlmywVsGgQecY@yamabiko.proxy.rlwy.net:57598/railway"

	var err error
	DB, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}

	if err := DB.Ping(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to PostgreSQL successfully!")

	// Run migrations
	if err := RunMigrations(); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}
}
