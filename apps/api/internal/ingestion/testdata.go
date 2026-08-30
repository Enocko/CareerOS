package ingestion

import "strings"

// TestExternalIDPrefixes identify opportunities created by automated tests.
// These must never appear in the student browse catalog.
var TestExternalIDPrefixes = []string{
	"API-TEST-",
	"UPSERT-TEST-",
	"FAIL-SYNC-TEST-",
	"STALE-SYNC-TEST-",
	"GH-FAIL-BOARD-",
	"GH-OK-BOARD-",
	"ASHBY-FAIL-BOARD-",
	"ASHBY-OK-BOARD-",
	"LEVER-FAIL-BOARD-",
	"LEVER-OK-BOARD-",
	"REU-UPSERT-TEST-",
}

// IsTestExternalID reports whether an external ID was created by automated tests.
func IsTestExternalID(externalID string) bool {
	for _, prefix := range TestExternalIDPrefixes {
		if strings.HasPrefix(externalID, prefix) {
			return true
		}
	}
	return false
}
