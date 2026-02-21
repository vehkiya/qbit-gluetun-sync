package watcher

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fsnotify/fsnotify"
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
		watcher.Close()
		return fmt.Errorf("failed to watch directory %s: %w", dir, err)
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// We care about CREATE or WRITE for our specific file
				if (event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) && event.Name == filePath {
					log.Printf("Detected event on port file: %s", event.Op)
					handleFileChange(filePath, callback)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Watcher error: %v", err)
			}
		}
	}()

	return nil
}

// CheckFileNow manually checks the file once, useful on startup.
func CheckFileNow(filePath string, callback func(port int)) {
	if _, err := os.Stat(filePath); err == nil {
		log.Printf("Initial check: found port file")
		handleFileChange(filePath, callback)
	}
}

// handleFileChange reads the file, parses the port, and executes the callback
func handleFileChange(filePath string, callback func(port int)) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Failed to read port file: %v", err)
		return
	}

	portStr := strings.TrimSpace(string(content))
	if portStr == "" {
		return
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Printf("Failed to parse port '%s': %v", portStr, err)
		return
	}

	if port <= 0 || port > 65535 {
		log.Printf("Invalid port number: %d", port)
		return
	}

	callback(port)
}
