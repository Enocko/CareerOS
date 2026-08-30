package ingestrecord

import (
	"encoding/json"
	"time"
)

// RawOpportunity is the normalized shape produced by source adapters.
type RawOpportunity struct {
	ExternalID      string
	Title           string
	Organization    string
	Description     string
	Category        string
	Location        string
	WorkArrangement string
	Deadline        *time.Time
	ApplicationURL  string
	SourceURL       string
	Compensation    string
	Tags            []string
	Skills          []string
	Eligibility     string
	// Universal schema fields (non-employment adapters set these explicitly).
	OpportunityType    string
	TypeMetadata       json.RawMessage
	VerificationMethod string
	// Relevance Engine v2 classification (employment ingestion only).
	ExperienceLevel       string
	CareerFamily          string
	EducationLevel        string
	RelevanceTier         string
	ClassificationReasons []string
}
