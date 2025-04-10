package main

import (
	"log/slog"
	"os"
)

// InitLogger initializes the JSON logger
func InitLogger() {
	// Create JSON handler with options
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo, // Set minimum level to log
		AddSource: true,           // Include source file and line in log
	})

	// Set the default logger
	slog.SetDefault(slog.New(jsonHandler))
}
