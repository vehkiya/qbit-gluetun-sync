package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/vehkiya/qbit-gluetun-sync/pkg/logger"
)

// WatchFile continuously watches a file's directory for CREATE or WRITE events
// and calls the provided callback with the port number when updated.
func WatchFile(filePath string, callback func(port int)) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	dir := filepath.Dir(filePath)
	err = watcher.Add(dir)
	if err != nil {
		_ = watcher.Close()
		return fmt.Errorf("failed to watch directory %s: %w", dir, err)
	}

	go func() {
		defer func() { _ = watcher.Close() }()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// We care about CREATE or WRITE for our specific file
				if (event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) && event.Name == filePath {
					logger.Debug("Detected event on port file: %s", "op", event.Op.String())
					handleFileChange(filePath, callback)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logger.Error("Watcher error", "err", err)
			}
		}
	}()

	return nil
}

// CheckFileNow manually checks the file once, useful on startup.
func CheckFileNow(filePath string, callback func(port int)) {
	if _, err := os.Stat(filePath); err == nil {
		logger.Debug("Initial check: found port file")
		handleFileChange(filePath, callback)
	}
}

// handleFileChange reads the file, parses the port, and executes the callback
func handleFileChange(filePath string, callback func(port int)) {
	//nolint:gosec // filePath is controlled by the environment and safe for read
	content, err := os.ReadFile(filePath)
	if err != nil {
		logger.Error("Failed to read port file", "err", err)
		return
	}

	portStr := strings.TrimSpace(string(content))
	if portStr == "" {
		return
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		logger.Error("Failed to parse port", "port", portStr, "err", err)
		return
	}

	if port <= 0 || port > 65535 {
		logger.Warn("Invalid port number", "port", port)
		return
	}

	callback(port)
}
