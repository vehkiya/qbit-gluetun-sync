# Go-Docker Template

This repository follows a specific pattern for containerizing Go scripts and tools by wrapping them in a lightweight Go-based proxy.

## Architecture Pattern

### 1. Go Entrypoint (`main.go`)
- **Lifecycle Management**: The Go program is the PID 1 inside the container. It handles initialization and manages event-driven tasks.
- **Reverse Proxy**: Uses `net/http/httputil` to provide a reverse proxy. This allows:
    - Adding custom endpoints (like `/healthz` or `/sync`).
    - Decoupling the target service from the exposed container port.
- **Event-Driven Tasks**: Implements background routines (e.g., watching a file for changes with `fsnotify`) to trigger actions instantly without heavy polling.
- **Environment Driven**: Configuration is strictly managed via environment variables with sensible defaults.

- **HTTP Testing**: Leverages `net/http/httptest` to verify the proxy and custom handlers.
- **Service Mocking**: Testing against mocked target services instead of requiring live dependencies.

### 3. Containerization (`Dockerfile`)
- **Multi-Stage Builds**:
    - `builder`: Compiles the Go entrypoint into a static, CGO-disabled binary.
    - `final`: Uses Google's `distroless/cc` (or similar minimal base) for security.
- **Rootless Security**:
    - Strictly runs as `USER nonroot:nonroot`.
    - No shell included in the final image (`distroless`).
- **Version Management**: Versions are automatically calculated using a semantic versioning (`semver`) strategy based on merged commits.

### 4. CI/CD (GitHub Actions)
- **Workflow Patterns**: Maintain separate workflows for `PR Validation` and `CD/Release`.
  - **PR Validation** (`pr.yaml`): Runs automatically on `pull_request` and `push` to `master`/`main` branches. It splits into two parallel jobs:
    - **Validate**: Sets up Go, runs `golangci-lint`, and executes tests (`go test -v -race ./...`).
    - **Security**: Builds the Docker image locally and runs `Trivy` to scan for OS and library vulnerabilities.
  - **CD / Release** (`cd.yaml`): Runs manually on `workflow_dispatch`. It calculates a semantic version, builds and pushes the Docker image to GitHub Container Registry (GHCR), generates a SLSA build provenance attestation, generates a changelog via `git log`, pushes a Git tag, and publishes a GitHub Release.

## Key Files
- `main.go`: The Go wrapper logic.
- `main_test.go`: Unit tests and HTTP mocks.
- `Dockerfile`: Multi-stage, rootless, distroless build.
- `.github/workflows/`: Automated validation and delivery.
- `renovate.json`: Automated dependency updates.
