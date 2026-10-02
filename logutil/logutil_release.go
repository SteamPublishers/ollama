//go:build !debug
// +build !debug

package logutil

import (
	"io"
	"log/slog"
	"os"
)

func GetLogLevel(in slog.Level) slog.Level {
	return slog.LevelError
}

func GetLogWriter() io.Writer {
	return os.Stderr
}
