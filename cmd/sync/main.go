package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/vehkiya/qbit-gluetun-sync/pkg/qbit"
	"github.com/vehkiya/qbit-gluetun-sync/pkg/watcher"
)

var currentPort int

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
		if port == currentPort {
			log.Printf("Port %d is already synced, skipping", port)
			return
		}
		log.Printf("Syncing new port %d to qBitTorrent", port)
		err := qbitClient.SetListenPort(port)
		if err != nil {
			log.Printf("Failed to set port: %v", err)
		} else {
			log.Printf("Successfully set port to %d", port)
			currentPort = port
		}
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

	// Setup HTTP Handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	mux.HandleFunc("/sync", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Manual sync requested")
		watcher.CheckFileNow(portFile, syncPortFunc)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Sync triggered"))
	})
	// Fallback to proxy
	mux.Handle("/", proxy)

	log.Printf("Starting reverse proxy on :%s proxying to %s", listenPort, qbitAddr)
	if err := http.ListenAndServe(":"+listenPort, mux); err != nil {
		log.Fatalf("Proxy server failed: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
