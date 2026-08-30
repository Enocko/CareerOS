package opportunitytype

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	maxMetadataStringLen = 500
	maxMetadataArrayLen  = 50
	maxMetadataKeyLen    = 64
)

// WriteInput is validated at opportunity write boundaries.
type WriteInput struct {
	OpportunityType    string
	ExperienceLevel    string
	EmploymentMode     string
	CareerFamily       string
	WorkArrangement    string
	TypeMetadata       json.RawMessage
	VerificationMethod string
}

// ValidateWrite enforces Model C dimension rules and type_metadata schemas.
func ValidateWrite(in WriteInput) error {
	if !ValidType(in.OpportunityType) {
		return fmt.Errorf("invalid opportunity_type: %q", in.OpportunityType)
	}

	if in.VerificationMethod != "" {
		if _, ok := validVerificationMethods[in.VerificationMethod]; !ok {
			return fmt.Errorf("invalid verification_method: %q", in.VerificationMethod)
		}
	}

	isEmployment := in.OpportunityType == Employment

	if isEmployment {
		if in.ExperienceLevel != "" {
			if _, ok := validExperienceLevels[in.ExperienceLevel]; !ok {
				return fmt.Errorf("invalid experience_level: %q", in.ExperienceLevel)
			}
		}
		if in.EmploymentMode != "" {
			if _, ok := validEmploymentModes[in.EmploymentMode]; !ok {
				return fmt.Errorf("invalid employment_mode: %q", in.EmploymentMode)
			}
		}
	} else {
		if in.ExperienceLevel != "" {
			return fmt.Errorf("experience_level is only valid for employment opportunities")
		}
		if in.EmploymentMode != "" {
			return fmt.Errorf("employment_mode is only valid for employment opportunities")
		}
		if in.CareerFamily != "" {
			return fmt.Errorf("career_family is only valid for employment opportunities in v1 writes")
		}
	}

	meta := in.TypeMetadata
	if len(meta) == 0 {
		meta = json.RawMessage(`{}`)
	}
	if !json.Valid(meta) {
		return fmt.Errorf("type_metadata must be valid JSON")
	}

	switch in.OpportunityType {
	case Employment:
		return validateEmploymentMetadata(meta)
	case Research:
		return validateResearchMetadata(meta)
	case Scholarship:
		return validateScholarshipMetadata(meta)
	case Fellowship:
		return validateFellowshipMetadata(meta)
	case Program, Event:
		return validateProgramEventMetadata(meta)
	case Competition:
		return validateCompetitionMetadata(meta)
	case Other:
		return validateEmptyOnlyMetadata(meta)
	default:
		return fmt.Errorf("unsupported opportunity_type: %q", in.OpportunityType)
	}
}

func validateEmploymentMetadata(raw json.RawMessage) error {
	return validateAllowedKeys(raw, nil)
}

func validateResearchMetadata(raw json.RawMessage) error {
	allowed := map[string]struct{}{
		"research_area": {}, "stipend": {}, "housing_provided": {}, "travel_support": {},
		"citizenship_required": {}, "duration_weeks": {}, "program_start": {}, "program_end": {},
		"program_url": {}, "application_status": {}, "application_status_method": {},
		"availability_verification_method": {}, "application_verified_at": {},
		"application_verification_source_url": {}, "cycle_label": {}, "opens_at": {},
		"next_verification_at": {},
	}
	if err := validateAllowedKeys(raw, allowed); err != nil {
		return err
	}
	var payload researchMetadata
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("invalid research type_metadata: %w", err)
	}
	return payload.validate()
}

func validateScholarshipMetadata(raw json.RawMessage) error {
	allowed := map[string]struct{}{
		"award_amount": {}, "award_amount_max": {}, "renewable": {}, "financial_need_required": {},
		"fields_of_study": {}, "hbcu_required": {},
	}
	if err := validateAllowedKeys(raw, allowed); err != nil {
		return err
	}
	var payload scholarshipMetadata
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("invalid scholarship type_metadata: %w", err)
	}
	return payload.validate()
}

func validateFellowshipMetadata(raw json.RawMessage) error {
	allowed := map[string]struct{}{
		"research_area": {}, "stipend": {}, "duration_weeks": {}, "program_start": {}, "program_end": {},
	}
	if err := validateAllowedKeys(raw, allowed); err != nil {
		return err
	}
	var payload fellowshipMetadata
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("invalid fellowship type_metadata: %w", err)
	}
	return payload.validate()
}

func validateProgramEventMetadata(raw json.RawMessage) error {
	allowed := map[string]struct{}{
		"program_format": {}, "event_subtype": {}, "event_start": {}, "event_end": {},
		"registration_cost": {}, "travel_support": {}, "target_class_years": {}, "paid": {}, "duration_weeks": {},
	}
	if err := validateAllowedKeys(raw, allowed); err != nil {
		return err
	}
	var payload programEventMetadata
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("invalid program/event type_metadata: %w", err)
	}
	return payload.validate()
}

func validateCompetitionMetadata(raw json.RawMessage) error {
	allowed := map[string]struct{}{
		"prize": {}, "team_size_max": {}, "submission_deadline": {},
	}
	if err := validateAllowedKeys(raw, allowed); err != nil {
		return err
	}
	var payload competitionMetadata
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("invalid competition type_metadata: %w", err)
	}
	return payload.validate()
}

func validateEmptyOnlyMetadata(raw json.RawMessage) error {
	return validateAllowedKeys(raw, map[string]struct{}{})
}

func validateAllowedKeys(raw json.RawMessage, allowed map[string]struct{}) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return fmt.Errorf("type_metadata must be a JSON object")
	}
	if len(obj) == 0 {
		return nil
	}
	if allowed == nil {
		for k := range obj {
			if len(k) > maxMetadataKeyLen {
				return fmt.Errorf("unsupported metadata key: %q", k)
			}
			return fmt.Errorf("unsupported metadata key: %q", k)
		}
		return nil
	}
	for k := range obj {
		if len(k) > maxMetadataKeyLen {
			return fmt.Errorf("unsupported metadata key: %q", k)
		}
		if _, ok := allowed[k]; !ok {
			return fmt.Errorf("unsupported metadata key: %q", k)
		}
	}
	return nil
}

type researchMetadata struct {
	ResearchArea                     *string `json:"research_area,omitempty"`
	Stipend                          *string `json:"stipend,omitempty"`
	HousingProvided                  *bool   `json:"housing_provided,omitempty"`
	TravelSupport                    *bool   `json:"travel_support,omitempty"`
	CitizenshipRequired              *string `json:"citizenship_required,omitempty"`
	DurationWeeks                    *int    `json:"duration_weeks,omitempty"`
	ProgramStart                     *string `json:"program_start,omitempty"`
	ProgramEnd                       *string `json:"program_end,omitempty"`
	ProgramURL                       *string `json:"program_url,omitempty"`
	ApplicationStatus                *string `json:"application_status,omitempty"`
	ApplicationStatusMethod          *string `json:"application_status_method,omitempty"`
	AvailabilityVerificationMethod   *string `json:"availability_verification_method,omitempty"`
	ApplicationVerifiedAt            *string `json:"application_verified_at,omitempty"`
	ApplicationVerificationSourceURL *string `json:"application_verification_source_url,omitempty"`
	CycleLabel                       *string `json:"cycle_label,omitempty"`
	OpensAt                          *string `json:"opens_at,omitempty"`
	NextVerificationAt               *string `json:"next_verification_at,omitempty"`
}

func (m researchMetadata) validate() error {
	if err := validateOptionalString(m.ResearchArea, "research_area"); err != nil {
		return err
	}
	if err := validateOptionalString(m.Stipend, "stipend"); err != nil {
		return err
	}
	if err := validateOptionalString(m.CitizenshipRequired, "citizenship_required"); err != nil {
		return err
	}
	if err := validateOptionalString(m.ProgramURL, "program_url"); err != nil {
		return err
	}
	if m.ApplicationStatus != nil && !ValidApplicationStatus(*m.ApplicationStatus) {
		return fmt.Errorf("invalid application_status: %q", *m.ApplicationStatus)
	}
	if m.ApplicationStatusMethod != nil && !ValidAvailabilityMethod(*m.ApplicationStatusMethod) {
		return fmt.Errorf("invalid application_status_method: %q", *m.ApplicationStatusMethod)
	}
	if m.AvailabilityVerificationMethod != nil && !ValidAvailabilityMethod(*m.AvailabilityVerificationMethod) {
		return fmt.Errorf("invalid availability_verification_method: %q", *m.AvailabilityVerificationMethod)
	}
	if err := validateOptionalString(m.ApplicationVerificationSourceURL, "application_verification_source_url"); err != nil {
		return err
	}
	if err := validateOptionalString(m.CycleLabel, "cycle_label"); err != nil {
		return err
	}
	if err := validateOptionalDate(m.OpensAt, "opens_at"); err != nil {
		return err
	}
	if err := validateOptionalString(m.ApplicationVerifiedAt, "application_verified_at"); err != nil {
		return err
	}
	if err := validateOptionalString(m.NextVerificationAt, "next_verification_at"); err != nil {
		return err
	}
	if m.DurationWeeks != nil && *m.DurationWeeks < 0 {
		return fmt.Errorf("duration_weeks must be non-negative")
	}
	if err := validateOptionalDate(m.ProgramStart, "program_start"); err != nil {
		return err
	}
	if err := validateOptionalDate(m.ProgramEnd, "program_end"); err != nil {
		return err
	}
	return nil
}

type scholarshipMetadata struct {
	AwardAmount           *string  `json:"award_amount,omitempty"`
	AwardAmountMax        *float64 `json:"award_amount_max,omitempty"`
	Renewable             *bool    `json:"renewable,omitempty"`
	FinancialNeedRequired *bool    `json:"financial_need_required,omitempty"`
	FieldsOfStudy         []string `json:"fields_of_study,omitempty"`
	HBCURequired          *bool    `json:"hbcu_required,omitempty"`
}

func (m scholarshipMetadata) validate() error {
	if err := validateOptionalString(m.AwardAmount, "award_amount"); err != nil {
		return err
	}
	if m.AwardAmountMax != nil && *m.AwardAmountMax < 0 {
		return fmt.Errorf("award_amount_max must be non-negative")
	}
	if len(m.FieldsOfStudy) > maxMetadataArrayLen {
		return fmt.Errorf("fields_of_study exceeds max length")
	}
	for _, f := range m.FieldsOfStudy {
		if len(strings.TrimSpace(f)) == 0 {
			return fmt.Errorf("fields_of_study entries must be non-empty")
		}
		if len(f) > maxMetadataStringLen {
			return fmt.Errorf("fields_of_study entry too long")
		}
	}
	return nil
}

type fellowshipMetadata struct {
	ResearchArea  *string `json:"research_area,omitempty"`
	Stipend       *string `json:"stipend,omitempty"`
	DurationWeeks *int    `json:"duration_weeks,omitempty"`
	ProgramStart  *string `json:"program_start,omitempty"`
	ProgramEnd    *string `json:"program_end,omitempty"`
}

func (m fellowshipMetadata) validate() error {
	if err := validateOptionalString(m.ResearchArea, "research_area"); err != nil {
		return err
	}
	if err := validateOptionalString(m.Stipend, "stipend"); err != nil {
		return err
	}
	if m.DurationWeeks != nil && *m.DurationWeeks < 0 {
		return fmt.Errorf("duration_weeks must be non-negative")
	}
	if err := validateOptionalDate(m.ProgramStart, "program_start"); err != nil {
		return err
	}
	if err := validateOptionalDate(m.ProgramEnd, "program_end"); err != nil {
		return err
	}
	return nil
}

type programEventMetadata struct {
	ProgramFormat    *string  `json:"program_format,omitempty"`
	EventSubtype     *string  `json:"event_subtype,omitempty"`
	EventStart       *string  `json:"event_start,omitempty"`
	EventEnd         *string  `json:"event_end,omitempty"`
	RegistrationCost *string  `json:"registration_cost,omitempty"`
	TravelSupport    *bool    `json:"travel_support,omitempty"`
	TargetClassYears []string `json:"target_class_years,omitempty"`
	Paid             *bool    `json:"paid,omitempty"`
	DurationWeeks    *int     `json:"duration_weeks,omitempty"`
}

func (m programEventMetadata) validate() error {
	if err := validateOptionalString(m.ProgramFormat, "program_format"); err != nil {
		return err
	}
	if err := validateOptionalString(m.EventSubtype, "event_subtype"); err != nil {
		return err
	}
	if err := validateOptionalString(m.RegistrationCost, "registration_cost"); err != nil {
		return err
	}
	if err := validateOptionalDate(m.EventStart, "event_start"); err != nil {
		return err
	}
	if err := validateOptionalDate(m.EventEnd, "event_end"); err != nil {
		return err
	}
	if m.EventStart != nil && m.EventEnd != nil {
		start, _ := time.Parse("2006-01-02", *m.EventStart)
		end, _ := time.Parse("2006-01-02", *m.EventEnd)
		if end.Before(start) {
			return fmt.Errorf("event_end must be on or after event_start")
		}
	}
	if len(m.TargetClassYears) > maxMetadataArrayLen {
		return fmt.Errorf("target_class_years exceeds max length")
	}
	for _, y := range m.TargetClassYears {
		if len(strings.TrimSpace(y)) == 0 {
			return fmt.Errorf("target_class_years entries must be non-empty")
		}
	}
	if m.DurationWeeks != nil && *m.DurationWeeks < 0 {
		return fmt.Errorf("duration_weeks must be non-negative")
	}
	return nil
}

type competitionMetadata struct {
	Prize              *string `json:"prize,omitempty"`
	TeamSizeMax        *int    `json:"team_size_max,omitempty"`
	SubmissionDeadline *string `json:"submission_deadline,omitempty"`
}

func (m competitionMetadata) validate() error {
	if err := validateOptionalString(m.Prize, "prize"); err != nil {
		return err
	}
	if m.TeamSizeMax != nil && *m.TeamSizeMax < 1 {
		return fmt.Errorf("team_size_max must be positive")
	}
	return validateOptionalDate(m.SubmissionDeadline, "submission_deadline")
}

func validateOptionalString(s *string, field string) error {
	if s == nil {
		return nil
	}
	if len(*s) > maxMetadataStringLen {
		return fmt.Errorf("%s exceeds max length", field)
	}
	return nil
}

func validateOptionalDate(s *string, field string) error {
	if s == nil || *s == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", *s); err != nil {
		return fmt.Errorf("%s must be YYYY-MM-DD", field)
	}
	return nil
}
