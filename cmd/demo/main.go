// Demo program showing SQLite IR code database usage
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/diogoaguiar/hvac-manager/internal/database"
)

func main() {
	ctx := context.Background()

	// Create in-memory database
	db, err := database.New(":memory:")
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Initialize schema
	fmt.Println("Initializing database schema...")
	if err := db.InitSchema(ctx); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	// Load IR codes from SmartIR reference files
	irCodesDir := "docs/smartir/reference"
	fmt.Printf("Loading IR codes from %s...\n", irCodesDir)

	err = db.LoadFromDirectory(ctx, irCodesDir)
	if err != nil {
		log.Fatalf("Failed to load IR codes: %v", err)
	}

	// List available models
	models, err := db.ListModels(ctx)
	if err != nil {
		log.Fatalf("Failed to list models: %v", err)
	}

	fmt.Printf("\n✓ Loaded %d AC models\n\n", len(models))

	// Demonstrate lookups for each model
	for _, modelID := range models {
		model, err := db.GetModel(ctx, modelID)
		if err != nil {
			log.Printf("Failed to get model %s: %v", modelID, err)
			continue
		}

		fmt.Printf("Model: %s (%s)\n", modelID, model.Manufacturer)
		fmt.Printf("  Temperature range: %d°C - %d°C\n", model.MinTemperature, model.MaxTemperature)

		// Lookup "off" command
		offCode, err := db.LookupOffCode(ctx, modelID)
		if err != nil {
			log.Printf("  No off code: %v", err)
		} else {
			fmt.Printf("  Off command: %s... (%d chars)\n", offCode[:min(20, len(offCode))], len(offCode))
		}

		// Lookup a cool mode command
		code, err := db.LookupCode(ctx, modelID, "cool", 21, "low")
		if err != nil {
			fmt.Printf("  Cool 21°C (low fan): not available\n")
		} else {
			fmt.Printf("  Cool 21°C (low fan): %s... (%d chars)\n", code[:min(20, len(code))], len(code))
		}

		fmt.Println()
	}

	// Interactive lookup example
	if len(os.Args) > 4 {
		modelID := os.Args[1]
		mode := os.Args[2]
		temp := os.Args[3]
		fan := os.Args[4]

		var tempInt int
		fmt.Sscanf(temp, "%d", &tempInt)

		fmt.Printf("\nLookup: Model=%s Mode=%s Temp=%d°C Fan=%s\n", modelID, mode, tempInt, fan)
		code, err := db.LookupCode(ctx, modelID, mode, tempInt, fan)
		if err != nil {
			log.Printf("Error: %v", err)
		} else {
			fmt.Printf("IR Code: %s\n", code)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
