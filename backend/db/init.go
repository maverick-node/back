package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
	migrate "github.com/rubenv/sql-migrate"
)

var DB *sql.DB

func Initdb() {
	var err error
	DB, err = sql.Open("sqlite3", "./db/db.db")
	if err != nil {
		fmt.Println("Failed to open database:", err)
		return
	}

	if err := DB.Ping(); err != nil {
		fmt.Println("Failed to connect to database:", err)
		return
	}

	migrations := &migrate.FileMigrationSource{
		Dir: "db/migrations/sqlite3",
	}

	n, err := migrate.Exec(DB, "sqlite3", migrations, migrate.Up)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	fmt.Printf("Applied %d migrations!\n", n)
}
