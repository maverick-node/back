package db

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"strings"
)

func RunMigrations() error {
	// Get all migration files
	migrationDir := filepath.Join("db", "migrations", "postgres")
	files, err := ioutil.ReadDir(migrationDir)
	if err != nil {
		return fmt.Errorf("error reading migration directory: %v", err)
	}

	// Sort files by name to ensure correct order
	var migrationFiles []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".up.sql") {
			migrationFiles = append(migrationFiles, file.Name())
		}
	}
	sort.Strings(migrationFiles)

	// Execute each migration in order
	for _, migrationFile := range migrationFiles {
		migrationPath := filepath.Join(migrationDir, migrationFile)
		sqlBytes, err := ioutil.ReadFile(migrationPath)
		if err != nil {
			return fmt.Errorf("error reading migration file %s: %v", migrationFile, err)
		}

		log.Printf("Executing migration: %s", migrationFile)
		_, err = DB.Exec(string(sqlBytes))
		if err != nil {
			return fmt.Errorf("error executing migration %s: %v", migrationFile, err)
		}
	}

	log.Println("All migrations completed successfully!")
	return nil
}
