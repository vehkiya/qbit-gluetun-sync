package main

import (
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/vehkiya/qbit-gluetun-sync/pkg/logger"
	"github.com/vehkiya/qbit-gluetun-sync/pkg/qbit"
	"github.com/vehkiya/qbit-gluetun-sync/pkg/watcher"
)

var (
	currentPort int
	portMu      sync.Mutex
)

func main() {
	logLevel := getEnv("LOG_LEVEL", "info")
	logger.Init(logLevel)

	// Parse environment variables
	qbitAddr := getEnv("QBIT_ADDR", "http://localhost:8080")
	qbitUser := getEnv("QBIT_USER", "")
	qbitPass := getEnv("QBIT_PASS", "")
	portFile := getEnv("PORT_FILE", "/tmp/gluetun/forwarded_port")
	listenPort := getEnv("LISTEN_PORT", "9090")

	// Initialize qBitTorrent Client
	qbitClient := qbit.NewClient(qbitAddr, qbitUser, qbitPass)

	// Callback to sync port
	syncPortFunc := func(port int) {
		portMu.Lock()
		defer portMu.Unlock()

		if port == currentPort {
			logger.Debug("Port is already synced, skipping", "port", port)
			return
		}

		logger.Info("Syncing new port to qBitTorrent", "port", port)

		var err error
		maxRetries := 5
		backoff := 1 * time.Second

		for i := 0; i < maxRetries; i++ {
			err = qbitClient.SetListenPort(port)
			if err == nil {
				logger.Info("Successfully set port", "port", port)
				currentPort = port
				return
			}

			logger.Warn("Failed to set port", "attempt", i+1, "maxRetries", maxRetries, "err", err)
			if i < maxRetries-1 {
				logger.Info("Retrying...", "backoff", backoff)
				time.Sleep(backoff)
				backoff *= 2
			}
		}

		logger.Error("Exhausted all retries. Failed to sync port to qBitTorrent", "port", port, "err", err)
	}

	// Do initial check in case file already exists
	watcher.CheckFileNow(portFile, syncPortFunc)

	// Start file watcher
	logger.Info("Starting watcher", "file", portFile)
	if err := watcher.WatchFile(portFile, syncPortFunc); err != nil {
		logger.Warn("Failed to start file watcher (will keep running without it)", "err", err)
	}

	mux := setupMux()

	logger.Info("Starting sidecar server", "listenPort", listenPort, "qbitAddr", qbitAddr)
	server := &http.Server{
		Addr:              ":" + listenPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		logger.Fatal("Server failed", "err", err)
	}
}

func setupMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			logger.Error("Failed to write healthz response", "err", err)
		}
	})
	return mux
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
