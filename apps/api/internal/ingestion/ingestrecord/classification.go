package ingestrecord

import v2 "github.com/careeros/api/internal/ingestion/relevance/v2"

// ApplyClassification populates v2 classification fields from title and description.
func (r *RawOpportunity) ApplyClassification() v2.Classification {
	fields := &v2.RawFields{
		ExperienceLevel:       r.ExperienceLevel,
		CareerFamily:          r.CareerFamily,
		EducationLevel:        r.EducationLevel,
		RelevanceTier:         r.RelevanceTier,
		ClassificationReasons: r.ClassificationReasons,
	}
	c := v2.ApplyToRaw(fields, r.Title, r.Description)
	r.ExperienceLevel = fields.ExperienceLevel
	r.CareerFamily = fields.CareerFamily
	r.EducationLevel = fields.EducationLevel
	r.RelevanceTier = fields.RelevanceTier
	r.ClassificationReasons = fields.ClassificationReasons
	return c
}
