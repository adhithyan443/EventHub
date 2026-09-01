// package logger

// import (
// 	"log/slog"
// 	"os"
// )

// func New() *slog.Logger {
// 	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
// 		Level: slog.LevelDebug,
// 	})

// 	return slog.New(handler)
// }

package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// New initializes and returns a slog.Logger writing to both stdout and a file.
// It also returns a cleanup function to close the log file descriptor.
func New(filePath string) (*slog.Logger, func() error, error) {
	// Ensure the destination directory exists
	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, nil, err
		}
	}

	// Open or create the log file in append mode
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666) //os.O_TRUNC || os.O_APPEND
	if err != nil {
		return nil, nil, err
	}

	// Combine stdout (terminal) and the file
	// (Use just `file` if you want logs ONLY in the file and NOT in the terminal)
	multiWriter := io.MultiWriter(os.Stdout, file)

	handler := slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	cleanup := func() error {
		return file.Close()
	}

	return slog.New(handler), cleanup, nil
}
