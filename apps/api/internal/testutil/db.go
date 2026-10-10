// Package testutil holds helpers shared by database-backed tests.
package testutil

import (
	"os"
	"testing"
)

// SkipIfNoDB skips a test whose database connection failed, so `go test`
// works on machines without PostgreSQL. In CI (CI=true, set by GitHub Actions)
// the database is required, and a failed connection fails the test instead of
// silently skipping it.
func SkipIfNoDB(t testing.TB, err error) {
	t.Helper()
	if os.Getenv("CI") == "true" {
		t.Fatalf("database not available in CI: %v", err)
	}
	t.Skipf("database not available: %v", err)
}
