package recommendation

import "time"

// PassesHardFilters applies conservative exclusion rules before scoring.
func PassesHardFilters(c Candidate, now time.Time) bool {
	if c.Status != "open" {
		return false
	}
	if c.VerificationStatus != "verified" {
		return false
	}
	if derefString(c.RelevanceTier) != "high_confidence_technical" {
		return false
	}
	if c.HasApplication {
		return false
	}
	if c.Deadline != nil {
		deadlineDay := time.Date(c.Deadline.Year(), c.Deadline.Month(), c.Deadline.Day(), 0, 0, 0, 0, time.UTC)
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		if deadlineDay.Before(today) {
			return false
		}
	}
	return true
}
