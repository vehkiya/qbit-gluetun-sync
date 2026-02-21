# Design Document: Gluetun-qBitTorrent Port Sync

## Overview
A lightweight, event-driven sidecar application written in Go to synchronize the dynamic forwarded port from Gluetun (ProtonVPN) to qBitTorrent.

**Goal:** Provide a robust, low-resource alternative to shell-script polling loops for Kubernetes environments.

## Architecture

### Components
1.  **File Watcher (fsnotify):** Monitors `/tmp/gluetun/forwarded_port` for `CREATE` or `WRITE` events.
2.  **qBitTorrent Client:** Validates connectivity to the qBitTorrent API and updates the listening port preference.
3.  **Resilience Layer:** Handles API unavailability (e.g., during startup) with exponential backoff retries.

### Workflow
1.  **Startup:**
    - Read configuration from environment variables.
    - Validate connection to qBitTorrent API (retry until successful).
    - Check if `forwarded_port` file already exists. If yes, read and sync immediately.
2.  **Event Loop:**
    - Start `fsnotify` watcher on the directory.
    - On `WRITE` or `CREATE` event for `forwarded_port`:
        - Read the new port number.
        - Deduplicate (ignore if port hasn't changed).
        - POST to qBitTorrent API: `/api/v2/app/setPreferences` with `json={"listen_port": <PORT>}`.
        - Log success/failure.

## Technical Specifications

### Language
- **Go (Golang)**: Chosen for static binary compilation, low memory footprint (~5-10MB), and robust concurrency primitives.

### Dependencies
- `github.com/fsnotify/fsnotify`: For cross-platform file system notifications.
- Standard `net/http`: For API communication.

### Configuration (Environment Variables)
| Variable | Default | Description |
| :--- | :--- | :--- |
| `QBIT_ADDR` | `http://localhost:8080` | Address of the qBitTorrent WebUI. |
| `QBIT_USER` | *(Optional)* | Username if auth bypass is disabled. |
| `QBIT_PASS` | *(Optional)* | Password if auth bypass is disabled. |
| `PORT_FILE` | `/tmp/gluetun/forwarded_port` | Path to the file created by Gluetun. |
| `LOG_LEVEL` | `info` | Logging verbosity (debug, info, error). |

## Implementation Plan

### Phase 1: Core Logic
- implement `main.go` with configuration parsing.
- Implement `port_watcher.go` using `fsnotify`.
- Implement `qbit_client.go` with `SetPreferences` method.

### Phase 2: Dockerization
- Multi-stage build (Builder -> Scratch/Distroless).
- **Base Image:** `gcr.io/distroless/static:nonroot` for security and minimal size.
- **Target Size:** < 15MB compressed.

### Phase 3: CI/CD
- GitHub Actions workflow to build and push to `ghcr.io`.
- Semantic versioning based on tags.

## Alternatives Considered
- **Shell Script (inotify-tools):** Requires a heavier base image (Alpine) and is harder to handle complex API retry logic/error handling robustly.
- **Python:** Heavier runtime and memory usage (~50MB+) compared to Go.

## Security
- **Non-Root:** The container must run as a non-root user (UID 65532 or similar).
- **ReadOnly Filesystem:** The application only needs read access to the volume and network access to localhost.
