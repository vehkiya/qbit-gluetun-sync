package watcher

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestWatchFile(t *testing.T) {
	tempDir := t.TempDir()
	portFile := filepath.Join(tempDir, "forwarded_port")

	// Create initial file
	err := os.WriteFile(portFile, []byte("11111\n"), 0600)
	if err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}

	var latestPort int32
	callback := func(port int) {
		//nolint:gosec // port numbers will never overflow int32
		atomic.StoreInt32(&latestPort, int32(port))
	}

	// Startup checks
	CheckFileNow(portFile, callback)
	time.Sleep(50 * time.Millisecond) // Let goroutine run if any

	if val := atomic.LoadInt32(&latestPort); val != 11111 {
		t.Fatalf("expected port 11111, got %d", val)
	}

	// Watch
	err = WatchFile(portFile, callback)
	if err != nil {
		t.Fatalf("WatchFile error: %v", err)
	}

	// Wait for watch to initialize
	time.Sleep(100 * time.Millisecond)

	// Simulate event
	atomic.StoreInt32(&latestPort, 0)
	err = os.WriteFile(portFile, []byte("22222\n"), 0600)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)
	if val := atomic.LoadInt32(&latestPort); val != 22222 {
		t.Fatalf("expected port 22222, got %d", val)
	}

	// Simulator bad event
	atomic.StoreInt32(&latestPort, 0)
	err = os.WriteFile(portFile, []byte("invalid\n"), 0600)
	if err != nil {
		t.Fatalf("failed to write invalid file: %v", err)
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)
	if val := atomic.LoadInt32(&latestPort); val != 0 {
		t.Fatalf("should not have triggered callback on invalid port, got %d", val)
	}
}
