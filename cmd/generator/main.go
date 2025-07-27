package main

import (
	"log"
	"os"

	"aireone.xyz/labtime/internal/apps/generator"
)

//go:generate go run ./main.go -output ../../labtime-configuration-schema.json

const (
	loggerPrefix = "generator:main:"
)

func main() {
	logger := log.New(os.Stdout, loggerPrefix, log.LstdFlags|log.Lshortfile)

	cfg := generator.LoadFlag()

	var schema string
	if err := generator.GenerateSchema(&schema); err != nil {
		logger.Fatalf("Error generating schema: %v", err)
	}

	if err := generator.WriteToFile(schema, cfg.GenerateSchemaFile); err != nil {
		logger.Fatalf("Error writing schema to file: %v", err)
	}
	logger.Printf("Schema successfully generated and written to %s", cfg.GenerateSchemaFile)
}
