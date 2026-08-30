package v2

import (
	"strings"
)

// EvaluateIngest classifies a posting and reports whether it should be persisted.
func EvaluateIngest(title, description string) (Classification, bool, string) {
	c := Classify(title, description)
	if !ShouldPersistSourceRecord(title, c) {
		return c, false, IngestFilterReason(title, c)
	}
	return c, true, ""
}

// ApplyToRaw copies classification fields onto a raw opportunity record.
func ApplyToRaw(raw *RawFields, title, description string) Classification {
	c := Classify(title, description)
	raw.ExperienceLevel = string(c.ExperienceLevel)
	raw.CareerFamily = string(c.CareerFamily)
	raw.EducationLevel = string(c.EducationLevel)
	raw.RelevanceTier = string(c.RelevanceTier)
	raw.ClassificationReasons = c.Reasons
	return c
}

// RawFields holds classification columns persisted with opportunities.
type RawFields struct {
	ExperienceLevel       string
	CareerFamily          string
	EducationLevel        string
	RelevanceTier         string
	ClassificationReasons []string
}

// ShouldPersistSourceRecord reports whether a posting should be stored during ingestion.
func ShouldPersistSourceRecord(title string, c Classification) bool {
	title = strings.TrimSpace(title)
	if title == "" {
		return false
	}
	if recruitingRolePattern.MatchString(title) &&
		!internPattern.MatchString(title) &&
		!coopPattern.MatchString(title) {
		return false
	}
	if SeniorTitleExclusion(title) {
		return false
	}
	return c.ExperienceLevel != ExperienceUnknown || hasEntryLevelEngineerTitle(title)
}
