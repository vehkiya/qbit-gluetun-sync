package main

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/vehkiya/qbit-gluetun-sync/pkg/logger"
	"github.com/vehkiya/qbit-gluetun-sync/pkg/netutil"
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
	allowedIPsStr := getEnv("ALLOWED_IPS", "127.0.0.1/32, ::1/128")

	allowedIPs, err := netutil.ParseAllowedIPs(allowedIPsStr)
	if err != nil {
		logger.Fatal("Invalid ALLOWED_IPS configuration", "err", err)
	}

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

	// Setup Reverse Proxy
	targetUrl, err := url.Parse(qbitAddr)
	if err != nil {
		logger.Fatal("Invalid QBIT_ADDR", "err", err)
	}
	proxy := httputil.NewSingleHostReverseProxy(targetUrl)

	mux := setupRouter(proxy, portFile, syncPortFunc, allowedIPs)

	logger.Info("Starting reverse proxy", "listenPort", listenPort, "qbitAddr", qbitAddr)
	server := &http.Server{
		Addr:              ":" + listenPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		logger.Fatal("Proxy server failed", "err", err)
	}
}

func setupRouter(proxy *httputil.ReverseProxy, portFile string, syncPortFunc func(int), allowedIPs []*net.IPNet) *http.ServeMux {
	mux := http.NewServeMux()

	// Healthz is always allowed without IP restrictions (useful for k8s/docker probes)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			logger.Error("Failed to write healthz response", "err", err)
		}
	})

	// Sync endpoint with IP restriction
	syncHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Manual sync requested")
		watcher.CheckFileNow(portFile, syncPortFunc)
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("Sync triggered")); err != nil {
			logger.Error("Failed to write sync response", "err", err)
		}
	})
	mux.Handle("/sync", netutil.IPAllowlistMiddleware(allowedIPs, syncHandler))

	// Fallback to proxy with IP restriction
	if proxy != nil {
		mux.Handle("/", netutil.IPAllowlistMiddleware(allowedIPs, proxy))
	}

	return mux
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
