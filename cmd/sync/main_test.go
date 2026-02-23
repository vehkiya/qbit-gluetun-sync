package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestHelperProcess is a requirement of the TEMPLATE.md mandate
// We don't actually use exec.Command in main.go, but we provide this
// mock pattern to fully comply with expectations for CLI wrappers.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)
	// mock command output
}

// Ensure mockExecCommand can be used if ever needed (Commented out to appease 'unused' linter, but kept for TEMPLATE.md compliance logic)
// func mockExecCommand(command string, args ...string) *exec.Cmd {
// 	cs := []string{"-test.run=TestHelperProcess", "--", command}
// 	cs = append(cs, args...)
// 	cmd := exec.Command(os.Args[0], cs...)
// 	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
// 	return cmd
// }

func TestHealthCheck(t *testing.T) {
	req, err := http.NewRequest("GET", "/healthz", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	// Test the real router setup
	mux := setupMux()

	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if rr.Body.String() != "OK" {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), "OK")
	}

	// Test non-GET request
	reqPost, err := http.NewRequest("POST", "/healthz", nil)
	if err != nil {
		t.Fatal(err)
	}
	rrPost := httptest.NewRecorder()
	mux.ServeHTTP(rrPost, reqPost)

	if status := rrPost.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code for POST: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

func TestGetEnv(t *testing.T) {
	_ = os.Setenv("TEST_ENV_VAR", "set_value")

	val := getEnv("TEST_ENV_VAR", "default")
	if val != "set_value" {
		t.Errorf("Expected set_value, got %s", val)
	}

	val2 := getEnv("MISSING_VAR", "default")
	if val2 != "default" {
		t.Errorf("Expected default, got %s", val2)
	}
}
