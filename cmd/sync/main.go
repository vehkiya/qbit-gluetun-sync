package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"

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
		if port == currentPort {
			portMu.Unlock()
			log.Printf("Port %d is already synced, skipping", port)
			return
		}
		portMu.Unlock()

		log.Printf("Syncing new port %d to qBitTorrent", port)
		err := qbitClient.SetListenPort(port)
		if err != nil {
			log.Printf("Failed to set port: %v", err)
		} else {
			log.Printf("Successfully set port to %d", port)
			portMu.Lock()
			currentPort = port
			portMu.Unlock()
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

	mux := setupRouter(proxy, portFile, syncPortFunc)

	log.Printf("Starting reverse proxy on :%s proxying to %s", listenPort, qbitAddr)
	//nolint:gosec // Basic proxy server without strict timeout requirements
	if err := http.ListenAndServe(":"+listenPort, mux); err != nil {
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
