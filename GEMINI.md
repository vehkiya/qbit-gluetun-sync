# Gemini CLI Project Mandates: Go-Docker CLI Wrapper

This project follows a specific architectural pattern for wrapping CLI tools in Go-based containers. Adhere strictly to these mandates to maintain design consistency and security.

## 1. Architectural Mandates

### Go Entrypoint (`main.go`)
- **Single Process Management**: The Go entrypoint must be the main process (`PID 1`).
- **Proxy Layer**: Use `httputil.NewSingleHostReverseProxy` for all requests to the wrapped CLI tool.
- **Custom Handlers**: Add `/healthz` for readiness/liveness probes and other operational endpoints (like `/sync`) as needed.
- **Child Processes**: Use `exec.Command` with the `mockExecCommand` pattern for any external CLI interaction to ensure testability.
- **Environment Variables**: Use `os.LookupEnv` or a similar pattern to handle environment-driven configuration with fallback defaults.

### Containerization (`Dockerfile`)
- **Multi-Stage Builds**:
  - Always use a `downloader` stage for external CLI binaries.
  - Always use a `builder` stage for the Go entrypoint.
  - Always use a `final` stage based on `gcr.io/distroless/cc-debian12` or similar minimal, shell-less images.
- **Rootless Execution**: The final image **MUST** run as `USER nonroot:nonroot`.
- **Static Binaries**: Compile the Go entrypoint with `CGO_ENABLED=0` to ensure compatibility with minimal base images.

## 2. Engineering Standards

### Testing & Validation
- **Empirical Reproduction**: Before fixing a bug, reproduce it with a test case in `main_test.go`.
- **Mocking**: Use the `TestHelperProcess` pattern to mock CLI output and exit codes.
- **No Side Effects**: Unit tests must not require the actual wrapped CLI binary or network access.
- **Linting**: All changes must pass `golangci-lint` as configured in the CI pipeline.

### CI/CD Consistency
- **Workflow Patterns**: Maintain separate workflows for `PR Validation` (test, build, scan) and `CD` (build, push, attest).
- **Vulnerability Scanning**: Use `Trivy` in the PR validation pipeline for OS and library-level scans.
- **Version Management**: Extract versions from `Dockerfile` `ARG`s to drive tagging and release metadata.

## 3. Style and Conventions
- **Go Version**: Keep the Go version in `go.mod` and `Dockerfile` synchronized.
- **Explicit Imports**: Group standard library imports separately from third-party ones.
- **Error Handling**: Wrap errors with context when bubble-up is necessary (e.g., `fmt.Errorf("context: %w", err)`).
- **Minimalism**: Avoid adding libraries for functionality that can be achieved with the Go standard library (e.g., simple HTTP proxying, periodic tickers).

## 4. Security
- **No Shell**: Do not include `sh`, `bash`, or other shells in the final image.
- **Credentials**: Never log sensitive environment variables (e.g., `BW_PASSWORD`, `BW_CLIENTSECRET`).
- **Static Analysis**: Maintain `golangci-lint` and `Trivy` as blocking gates in CI.
