package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/vehkiya/qbit-gluetun-sync/pkg/qbit"
	"github.com/vehkiya/qbit-gluetun-sync/pkg/watcher"
)

var (
	currentPort int
	portMu      sync.Mutex
)

func main() {
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
			log.Printf("Port %d is already synced, skipping", port)
			return
		}

		log.Printf("Syncing new port %d to qBitTorrent", port)

		var err error
		maxRetries := 5
		backoff := 1 * time.Second

		for i := 0; i < maxRetries; i++ {
			err = qbitClient.SetListenPort(port)
			if err == nil {
				log.Printf("Successfully set port to %d", port)
				currentPort = port
				return
			}

			log.Printf("Failed to set port (attempt %d/%d): %v", i+1, maxRetries, err)
			if i < maxRetries-1 {
				log.Printf("Retrying in %v...", backoff)
				time.Sleep(backoff)
				backoff *= 2
			}
		}

		log.Printf("Exhausted all retries. Failed to sync port %d to qBitTorrent: %v", port, err)
	}

	// Do initial check in case file already exists
	watcher.CheckFileNow(portFile, syncPortFunc)

	// Start file watcher
	log.Printf("Starting watcher for %s", portFile)
	err := watcher.WatchFile(portFile, syncPortFunc)
	if err != nil {
		log.Printf("Failed to start file watcher (will keep running without it): %v", err)
	}

	// Setup Reverse Proxy
	targetUrl, err := url.Parse(qbitAddr)
	if err != nil {
		log.Fatalf("Invalid QBIT_ADDR: %v", err)
	}
	proxy := httputil.NewSingleHostReverseProxy(targetUrl)

	mux := setupRouter(proxy, portFile, syncPortFunc)

	log.Printf("Starting reverse proxy on :%s proxying to %s", listenPort, qbitAddr)
	server := &http.Server{
		Addr:              ":" + listenPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Proxy server failed: %v", err)
	}
}

func setupRouter(proxy *httputil.ReverseProxy, portFile string, syncPortFunc func(int)) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			log.Printf("Failed to write healthz response: %v", err)
		}
	})
	mux.HandleFunc("/sync", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Manual sync requested")
		watcher.CheckFileNow(portFile, syncPortFunc)
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("Sync triggered")); err != nil {
			log.Printf("Failed to write sync response: %v", err)
		}
	})
	// Fallback to proxy
	mux.Handle("/", proxy)
	return mux
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
