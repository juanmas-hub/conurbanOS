package utils

import (
	"io"
	"log"
	"log/slog"
	"os"
	"strings"
)

func ConfigurarLogger(filePath string) {
	logFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}

func Log_level_from_string(string_level string) slog.Level {
	switch strings.ToUpper(string_level) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default: // TODO: esto hay que cambiarlo
		return slog.LevelInfo
	}
}
