//go:build debug
// +build debug

package logutil

import (
	"io"
	"log/slog"
	"math"
	"os"
)

func GetLogLevel(in slog.Level) slog.Level {
	return slog.Level(math.Max(int(slog.LevelInfo), int(int)))
}

func GetLogWriter() io.Writer {
	return os.Stderr
}
