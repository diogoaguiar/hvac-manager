// Database management CLI tool
// Build tags: dbtools
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/diogoaguiar/hvac-manager/internal/database"
)

func main() {
	if len(os.Args) < 3 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	dbPath := os.Args[2]
	ctx := context.Background()

	switch command {
	case "init":
		initDB(ctx, dbPath)
	case "load":
		if len(os.Args) < 4 {
			fmt.Println("Error: load command requires directory path")
			printUsage()
			os.Exit(1)
		}
		loadDB(ctx, dbPath, os.Args[3])
	case "load-single":
		if len(os.Args) < 5 {
			fmt.Println("Error: load-single command requires model ID and file path")
			printUsage()
			os.Exit(1)
		}
		loadSingleFile(ctx, dbPath, os.Args[3], os.Args[4])
	case "status":
		statusDB(ctx, dbPath)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: db <command> <database-file> [args]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  init <db-file>                    - Initialize database schema")
	fmt.Println("  load <db-file> <dir>              - Load IR codes from directory")
	fmt.Println("  load-single <db-file> <id> <file> - Load single SmartIR file with model ID")
	fmt.Println("  status <db-file>                  - Show database status")
	fmt.Println("")
	fmt.Println("The loader automatically detects and converts Broadlink format to Tuya.")
}

func initDB(ctx context.Context, dbPath string) {
	db, err := database.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.InitSchema(ctx); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	fmt.Printf("✓ Database initialized: %s\n", dbPath)
}

func loadDB(ctx context.Context, dbPath, dirPath string) {
	db, err := database.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Ensure schema is initialized
	if err := db.Migrate(ctx); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	if err := db.LoadFromDirectory(ctx, dirPath); err != nil {
		log.Fatalf("Failed to load IR codes: %v", err)
	}

	models, err := db.ListModels(ctx)
	if err != nil {
		log.Fatalf("Failed to list models: %v", err)
	}

	fmt.Printf("✓ Loaded %d models from %s\n", len(models), dirPath)
	for _, modelID := range models {
		fmt.Printf("  - %s\n", modelID)
	}
}

func loadSingleFile(ctx context.Context, dbPath, modelID, filePath string) {
	db, err := database.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Ensure schema is initialized
	if err := db.Migrate(ctx); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Load the single file
	if err := db.LoadFromJSON(ctx, modelID, filePath); err != nil {
		log.Fatalf("Failed to load file: %v", err)
	}

	fmt.Printf("✓ Loaded model %s from %s\n", modelID, filePath)
}

func statusDB(ctx context.Context, dbPath string) {
	db, err := database.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Get schema version
	version, err := db.GetSchemaVersion(ctx)
	if err != nil {
		fmt.Printf("Schema version: Error - %v\n", err)
	} else {
		fmt.Printf("Schema version: %d\n", version)
		if version == 0 {
			fmt.Println("Status: Uninitialized (run 'make db-init')")
			return
		}
	}

	// List models
	models, err := db.ListModels(ctx)
	if err != nil {
		fmt.Printf("Error listing models: %v\n", err)
		return
	}

	fmt.Printf("Models loaded: %d\n", len(models))
	for _, modelID := range models {
		model, err := db.GetModel(ctx, modelID)
		if err != nil {
			fmt.Printf("  - %s (error: %v)\n", modelID, err)
			continue
		}
		fmt.Printf("  - %s (%s, %d°C-%d°C)\n", modelID, model.Manufacturer, model.MinTemperature, model.MaxTemperature)
	}
}
