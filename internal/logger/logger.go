package logger

import (
	"log/slog"
	"os"
)

func InitLogger() {
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	})
	slog.SetDefault(slog.New(jsonHandler))
}
