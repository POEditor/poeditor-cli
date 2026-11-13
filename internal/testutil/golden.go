package testutil

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// UpdateEnv set to 1 rewrites golden files: UPDATE_GOLDEN=1 go test ./...
// (an env var, not a -update flag: go test ./... would pass the flag to
// packages that never define it and fail them).
const UpdateEnv = "UPDATE_GOLDEN"

var unsafeChars = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

// Golden compares got with testdata/<test name>.golden. Run the tests with
// UPDATE_GOLDEN=1 to (re)write the file, then review the diff before committing.
// Line endings are normalised so the files compare equal on Windows.
func Golden(t testing.TB, got string) {
	t.Helper()
	name := unsafeChars.ReplaceAllString(t.Name(), "_")
	path := filepath.Join("testdata", name+".golden")
	got = normalise(got)

	if os.Getenv(UpdateEnv) == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden file %s missing; run `UPDATE_GOLDEN=1 go test ./...` and review it: %v", path, err)
	}
	if normalise(string(want)) != got {
		t.Errorf("output differs from %s\n--- want ---\n%s\n--- got ---\n%s", path, want, got)
	}
}

func normalise(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}
