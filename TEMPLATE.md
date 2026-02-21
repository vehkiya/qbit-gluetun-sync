# Go-Docker CLI Wrapper Template

This repository follows a specific pattern for containerizing CLI tools by wrapping them in a lightweight Go-based proxy and lifecycle manager.

## Architecture Pattern

### 1. Go Entrypoint (`main.go`)
- **Lifecycle Management**: The Go program is the PID 1 inside the container. It handles initialization (e.g., login/unlock of the wrapped CLI) and manages child processes.
- **Reverse Proxy**: Uses `net/http/httputil` to provide a reverse proxy. This allows:
    - Adding custom endpoints (like `/healthz` or `/sync`).
    - Decoupling the internal CLI port from the exposed container port.
- **Periodic Tasks**: Implements background routines (e.g., periodic synchronization) using Go tickers and internal HTTP requests to its own endpoints.
- **Environment Driven**: Configuration is strictly managed via environment variables with sensible defaults.

### 2. Testing Strategy
- **Mocking External Commands**: Uses the `TestHelperProcess` pattern to mock `exec.Command` calls, allowing unit tests to run without the actual CLI binary present.
- **HTTP Testing**: Leverages `net/http/httptest` to verify the proxy and custom handlers.

### 3. Containerization (`Dockerfile`)
- **Multi-Stage Builds**:
    - `downloader`: Specialized stage for fetching external binaries (e.g., Bitwarden CLI).
    - `builder`: Compiles the Go entrypoint into a static, CGO-disabled binary.
    - `final`: Uses Google's `distroless/cc` (or similar minimal base) for security.
- **Rootless Security**:
    - Strictly runs as `USER nonroot:nonroot`.
    - No shell included in the final image (`distroless`).
- **Version Management**: The `BW_CLI_VERSION` is defined as an `ARG` in the Dockerfile and extracted by CI for tagging.

### 4. CI/CD (GitHub Actions)
- **PR Validation**:
    - Linting with `golangci-lint`.
    - Unit testing with `go test`.
    - Vulnerability scanning with `Trivy`.
- **Build & Push**:
    - Pushes to GitHub Container Registry (GHCR).
    - Automates tagging based on the version extracted from the Dockerfile.
    - Generates build attestations and provenance.

## Key Files
- `main.go`: The Go wrapper logic.
- `main_test.go`: Unit tests and `exec` mocks.
- `Dockerfile`: Multi-stage, rootless, distroless build.
- `.github/workflows/`: Automated validation and delivery.
- `renovate.json`: Automated dependency updates.
