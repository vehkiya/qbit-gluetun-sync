# Gemini CLI Project Mandates: Go-Docker Port Sync Sidecar

This project follows a specific architectural pattern for containerizing event-driven Go scripts and tools, specifically functioning as a background sidecar. Adhere strictly to these mandates to maintain design consistency and security.

## 1. Architectural Mandates

### Go Entrypoint (`main.go`)
- **Single Process Management**: The Go entrypoint must be the main process (`PID 1`).
- **No Proxy Layer**: The application functions purely as a background sidecar, without intercepting or proxying traffic to the target service.
- **Custom Handlers**: Add `/healthz` for readiness/liveness probes and other operational endpoints as needed.
- **Event-Driven Tasks**: Implement background routines (e.g., watching a file for changes) to trigger actions instantly without heavy polling.
- **Environment Variables**: Use `os.LookupEnv` or a similar pattern to handle environment-driven configuration with fallback defaults.

### Containerization (`Dockerfile`)
- **Multi-Stage Builds**:
  - Always use a `builder` stage for the Go entrypoint.
  - Always use a `final` stage based on `gcr.io/distroless/cc-debian12` or similar minimal, shell-less images.
- **Rootless Execution**: The final image **MUST** run as `USER nonroot:nonroot`.
- **Static Binaries**: Compile the Go entrypoint with `CGO_ENABLED=0` to ensure compatibility with minimal base images.

## 2. Engineering Standards

### Testing & Validation
- **Empirical Reproduction**: Before fixing a bug, reproduce it with a test case in `main_test.go`.
- **Mocking**: Use HTTP mocking tools like `httptest` to test against mocked target services.
- **No Side Effects**: Unit tests must not require live target services or network access.
- **Linting**: All changes must pass `golangci-lint` as configured in the CI pipeline.

### CI/CD Consistency
- **Workflow Patterns**: Maintain separate workflows for `PR Validation` and `CD/Release`.
  - **PR Validation** (`pr.yaml`): Runs automatically on `pull_request` and `push` to `master`/`main` branches. It splits into two parallel jobs:
    - **Validate**: Sets up Go, runs `golangci-lint`, and executes tests (`go test -v -race ./...`).
    - **Security**: Builds the Docker image locally and runs `Trivy` to scan for OS and library vulnerabilities.
  - **CD / Release** (`cd.yaml`): Runs manually on `workflow_dispatch`. It calculates a semantic version, builds and pushes the Docker image to GitHub Container Registry (GHCR), generates a SLSA build provenance attestation, generates a changelog via `git log`, pushes a Git tag, and publishes a GitHub Release.
- **Vulnerability Scanning**: Use `Trivy` in the PR validation pipeline. Must fail the pipeline (`exit-code: "1"`) on `CRITICAL` or `HIGH` severity vulnerabilities.
- **Version Management**:
  - Dynamically calculate semantic versions (`${major}.${minor}.${patch}`) from commits since the last tag.
  - Generate a changelog by listing commits since the previous tag (`git log ... --pretty=format:"* %s (%h)"`).
  - Use the generated changelog and calculated version to automatically push a Git tag and publish a GitHub Release.
## 3. Style and Conventions
- **Go Version**: Keep the Go version in `go.mod` and `Dockerfile` synchronized.
- **Explicit Imports**: Group standard library imports separately from third-party ones.
- **Error Handling**: Wrap errors with context when bubble-up is necessary (e.g., `fmt.Errorf("context: %w", err)`).
- **Minimalism**: Avoid adding libraries for functionality that can be achieved with the Go standard library (e.g., simple HTTP servers, periodic tickers).

## 4. Security
- **No Shell**: Do not include `sh`, `bash`, or other shells in the final image.
- **Credentials**: Never log sensitive environment variables (e.g., passwords or tokens).
- **Static Analysis**: Maintain `golangci-lint` and `Trivy` as blocking gates in CI.
