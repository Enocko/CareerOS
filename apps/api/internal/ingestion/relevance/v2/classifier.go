package v2

import (
	"regexp"
	"strings"
)

// Filter reason constants for ingestion diagnostics.
const (
	FilterReasonEmptyTitle              = "empty_title"
	FilterReasonNoStudentSignal         = "no_student_experience_signal"
	FilterReasonSeniorTitleExclusion    = "senior_title_exclusion"
	FilterReasonRecruitingRoleExclusion = "recruiting_role_exclusion"
	FilterReasonNotListedExclusion      = "not_listed_exclusion"
)

var (
	internPattern = regexp.MustCompile(`(?i)\bintern(ship)?\b`)
	coopPattern   = regexp.MustCompile(`(?i)\bco-?op\b`)
	newGradPattern = regexp.MustCompile(`(?i)\bnew\s?grad(uate)?\b`)
	earlyCareerPattern = regexp.MustCompile(`(?i)\bearly\s+career\b`)
	campusPattern = regexp.MustCompile(`(?i)\bcampus\b`)
	apprenticePattern = regexp.MustCompile(`(?i)\bapprentice(ship)?\b`)
	fellowPattern = regexp.MustCompile(`(?i)\bfellow(ship)?\b`)

	entryLevelEngineerPattern = regexp.MustCompile(`(?i)\b(associate\s+software|graduate\s+software|university\s+graduate|entry[\s-]level\s+(engineer|developer)|developer\s+intern|research\s+engineer|applied\s+scientist|technology\s+analyst)\b`)
	softwareEngineerILevelPattern = regexp.MustCompile(`(?i)\bsoftware\s+engineer\s+i\b`)
	seniorLevelEngineerIPattern = regexp.MustCompile(`(?i)\b(staff|principal|senior)\b.*\bengineer\s+i\b`)

	seniorPattern = regexp.MustCompile(`(?i)\b(senior|staff|principal|distinguished|director|vice\s+president|\bvp\b)\b`)

	recruitingRolePattern = regexp.MustCompile(`(?i)\b(university\s+recruiter|campus\s+recruiter|talent\s+acquisition|recruiting\s+manager)\b`)

	phdPattern = regexp.MustCompile(`(?i)\b(ph\.?\s*d\.?|doctorate|doctoral)\b`)
	mastersPattern = regexp.MustCompile(`(?i)\b(master'?s|m\.?\s*s\.?\b|mba)\b`)
	undergradPattern = regexp.MustCompile(`(?i)\b(undergraduate|undergrad|bachelor'?s)\b`)
	graduatePattern = regexp.MustCompile(`(?i)\bgraduate\b`)

	nonTechnicalRules = []struct {
		subfamily string
		pattern   *regexp.Regexp
	}{
		{"sales", regexp.MustCompile(`(?i)\b(sdr\b|sales\s+development\s+representative|sales\s+intern\b)\b`)},
		{"sales", regexp.MustCompile(`(?i)\baccount\s+executive\b`)},
		{"marketing", regexp.MustCompile(`(?i)\bproduct\s+marketing\b`)},
		{"marketing", regexp.MustCompile(`(?i)\bmarketing\s+intern\b`)},
		{"marketing", regexp.MustCompile(`(?i)\bmarketing\b`)},
		{"communications", regexp.MustCompile(`(?i)\bcommunications\s+intern\b`)},
		{"communications", regexp.MustCompile(`(?i)\bcommunications\b`)},
		{"recruiting", regexp.MustCompile(`(?i)\brecruiting\s+intern\b`)},
		{"recruiting", regexp.MustCompile(`(?i)\brecruiter\b`)},
		{"hr", regexp.MustCompile(`(?i)\b(human\s+resources|hr\s+intern\b)\b`)},
		{"customer_experience", regexp.MustCompile(`(?i)\bcustomer\s+experience\b`)},
		{"customer_success", regexp.MustCompile(`(?i)\bcustomer\s+success\b`)},
		{"business_development", regexp.MustCompile(`(?i)\b(business\s+development|\bbdr\b)\b`)},
		{"legal", regexp.MustCompile(`(?i)\blegal\s+intern\b`)},
		{"finance", regexp.MustCompile(`(?i)\b(finance\s+intern\b|accounting\s+intern\b)\b`)},
	}

	technicalRules = []struct {
		family  CareerFamily
		pattern *regexp.Regexp
	}{
		{CareerMachineLearningAI, regexp.MustCompile(`(?i)\b(machine\s+learning|ml\s+engineer|ai\s+engineer|applied\s+scientist)\b`)},
		{CareerDataScience, regexp.MustCompile(`(?i)\b(data\s+scientist|data\s+science)\b`)},
		{CareerDataScience, regexp.MustCompile(`(?i)\bdata\s+engineer\b`)},
		{CareerCybersecurity, regexp.MustCompile(`(?i)\b(cyber\s*security|cybersecurity|security\s+engineer|security\s+engineering|information\s+security)\b`)},
		{CareerCloudInfrastructureDevOps, regexp.MustCompile(`(?i)\b(devops|site\s+reliability|sre\b|cloud\s+engineer|infrastructure\s+engineer|platform\s+engineer)\b`)},
		{CareerProductManagementTechnical, regexp.MustCompile(`(?i)\btechnical\s+product\b`)},
		{CareerProductManagementTechnical, regexp.MustCompile(`(?i)\bproduct\s+management\s+intern\b`)},
		{CareerProductManagementTechnical, regexp.MustCompile(`(?i)\bproduct\s+manager\s+intern\b`)},
		{CareerQuantitativeTechnology, regexp.MustCompile(`(?i)\b(quantitative\s+(developer|researcher)|quant\s+developer)\b`)},
		{CareerTechnicalResearch, regexp.MustCompile(`(?i)\bresearch\s+engineer\b`)},
		{CareerTechnicalResearch, regexp.MustCompile(`(?i)\bresearch\s+intern\b`)},
		{CareerSoftwareEngineering, regexp.MustCompile(`(?i)\b(software\s+engineer|software\s+developer|software\s+engineering)\b`)},
		{CareerSoftwareEngineering, regexp.MustCompile(`(?i)\b(backend|front[\s-]?end|full[\s-]?stack|mobile\s+engineer)\b`)},
		{CareerSoftwareEngineering, regexp.MustCompile(`(?i)\bdeveloper\s+intern\b`)},
		{CareerSoftwareEngineering, regexp.MustCompile(`(?i)\b(associate\s+software|graduate\s+software|entry[\s-]level\s+(engineer|developer))\b`)},
		{CareerSoftwareEngineering, regexp.MustCompile(`(?i)\bsoftware\s+engineer\s+i\b`)},
		{CareerSoftwareEngineering, regexp.MustCompile(`(?i)\bengineering\s+co-?op\b`)},
		{CareerSoftwareEngineering, regexp.MustCompile(`(?i)\bengineering\s+intern\b`)},
		{CareerOtherTechnical, regexp.MustCompile(`(?i)\btechnology\s+analyst\b`)},
	}
)

// Classify applies deterministic title and description rules.
func Classify(title, description string) Classification {
	title = strings.TrimSpace(title)
	text := normalizeText(title, description)

	if title == "" {
		return Classification{
			ExperienceLevel: ExperienceUnknown,
			CareerFamily:    CareerUnknown,
			EducationLevel:  EducationUnspecified,
			RelevanceTier:   TierAmbiguous,
			Reasons:         []string{"empty_title"},
		}
	}

	exp := detectExperienceLevel(title)
	edu := detectEducationLevel(text)
	family, familyReasons := detectCareerFamily(title)
	reasons := append([]string{}, familyReasons...)

	if exp != ExperienceUnknown {
		reasons = append([]string{string(exp)}, reasons...)
	} else if hasEntryLevelEngineerTitle(title) {
		reasons = append([]string{"early_career_entry_level_engineer"}, reasons...)
	}

	if edu != EducationUnspecified {
		reasons = append(reasons, string(edu))
	}

	tier, inFeed := assignTier(exp, family, edu, title)
	if tier == TierAmbiguous && family == CareerUnknown {
		reasons = append(reasons, "unknown_career_family")
	}
	if edu == EducationPhD && (exp == ExperienceInternship || exp == ExperienceFellowship) {
		reasons = append(reasons, "phd_research_internship")
	}

	return Classification{
		ExperienceLevel: exp,
		CareerFamily:    family,
		EducationLevel:  edu,
		RelevanceTier:   tier,
		Reasons:         dedupeReasons(reasons),
		InTechnicalFeed: inFeed,
	}
}

func normalizeText(title, description string) string {
	desc := description
	if len(desc) > 2000 {
		desc = desc[:2000]
	}
	return strings.ToLower(title + " " + desc)
}

func detectExperienceLevel(title string) ExperienceLevel {
	switch {
	case internPattern.MatchString(title):
		return ExperienceInternship
	case coopPattern.MatchString(title):
		return ExperienceCoOp
	case newGradPattern.MatchString(title):
		return ExperienceNewGrad
	case apprenticePattern.MatchString(title):
		return ExperienceApprenticeship
	case fellowPattern.MatchString(title):
		return ExperienceFellowship
	case earlyCareerPattern.MatchString(title) || campusPattern.MatchString(title):
		return ExperienceEarlyCareer
	case hasEntryLevelEngineerTitle(title):
		return ExperienceEarlyCareer
	default:
		return ExperienceUnknown
	}
}

func detectEducationLevel(text string) EducationLevel {
	switch {
	case phdPattern.MatchString(text):
		return EducationPhD
	case mastersPattern.MatchString(text):
		return EducationMasters
	case undergradPattern.MatchString(text):
		return EducationUndergraduate
	case graduatePattern.MatchString(text) && !newGradPattern.MatchString(text):
		return EducationGraduateAny
	default:
		return EducationUnspecified
	}
}

func detectCareerFamily(title string) (CareerFamily, []string) {
	if isSalesNonTechnical(title) {
		return CareerNonTechnical, []string{"non_technical_sales"}
	}

	for _, rule := range nonTechnicalRules {
		if rule.pattern.MatchString(title) {
			return CareerNonTechnical, []string{"non_technical_" + rule.subfamily}
		}
	}

	for _, rule := range technicalRules {
		if rule.pattern.MatchString(title) {
			return rule.family, []string{string(rule.family)}
		}
	}

	return CareerUnknown, nil
}

func isSalesNonTechnical(title string) bool {
	lower := strings.ToLower(title)
	if strings.Contains(lower, "salesforce") || strings.Contains(lower, "sales engineer") {
		return false
	}
	if regexp.MustCompile(`(?i)\b(sdr\b|sales\s+development\s+representative|sales\s+intern\b)\b`).MatchString(title) {
		return true
	}
	if regexp.MustCompile(`(?i)\bsales\b`).MatchString(title) && !regexp.MustCompile(`(?i)\b(engineer|developer|scientist)\b`).MatchString(title) {
		return true
	}
	return false
}

func assignTier(exp ExperienceLevel, family CareerFamily, edu EducationLevel, title string) (RelevanceTier, bool) {
	hasStudentSignal := exp != ExperienceUnknown || hasEntryLevelEngineerTitle(title)

	if family == CareerNonTechnical && hasStudentSignal {
		return TierHighConfidenceNonTechnical, false
	}

	if isTechnicalFamily(family) && hasStudentSignal {
		if edu == EducationPhD && (exp == ExperienceInternship || exp == ExperienceFellowship) {
			return TierAmbiguous, false
		}
		return TierHighConfidenceTechnical, true
	}

	if hasStudentSignal {
		return TierAmbiguous, false
	}

	return TierAmbiguous, false
}

func isTechnicalFamily(f CareerFamily) bool {
	switch f {
	case CareerSoftwareEngineering, CareerDataScience, CareerMachineLearningAI,
		CareerCybersecurity, CareerProductManagementTechnical,
		CareerCloudInfrastructureDevOps, CareerQuantitativeTechnology,
		CareerTechnicalResearch, CareerOtherTechnical:
		return true
	default:
		return false
	}
}

// SeniorTitleExclusion checks senior titles without student/entry-level modifiers.
func SeniorTitleExclusion(title string) bool {
	if !seniorPattern.MatchString(title) {
		return false
	}
	if internPattern.MatchString(title) || coopPattern.MatchString(title) ||
		newGradPattern.MatchString(title) || hasEntryLevelEngineerTitle(title) {
		return false
	}
	return true
}

func hasEntryLevelEngineerTitle(title string) bool {
	if seniorLevelEngineerIPattern.MatchString(title) {
		return false
	}
	if softwareEngineerILevelPattern.MatchString(title) {
		return true
	}
	return entryLevelEngineerPattern.MatchString(title)
}

// IngestFilterReason returns a diagnostic reason when a posting is not persisted.
func IngestFilterReason(title string, c Classification) string {
	if strings.TrimSpace(title) == "" {
		return FilterReasonEmptyTitle
	}
	if recruitingRolePattern.MatchString(title) && !internPattern.MatchString(title) {
		return FilterReasonRecruitingRoleExclusion
	}
	if SeniorTitleExclusion(title) {
		return FilterReasonSeniorTitleExclusion
	}
	return FilterReasonNoStudentSignal
}

// PrimaryReasonCode returns a compact explainability code for diagnostics.
func (c Classification) PrimaryReasonCode() string {
	if len(c.Reasons) == 0 {
		return "unknown"
	}
	exp := string(c.ExperienceLevel)
	if c.ExperienceLevel == ExperienceUnknown {
		exp = "unknown_experience"
	}
	family := string(c.CareerFamily)
	if c.CareerFamily == CareerNonTechnical && len(c.Reasons) > 0 {
		for _, r := range c.Reasons {
			if strings.HasPrefix(r, "non_technical_") {
				return exp + " + " + r
			}
		}
	}
	if c.CareerFamily == CareerUnknown {
		return exp + " + unknown_career_family"
	}
	return exp + " + " + family
}

func dedupeReasons(reasons []string) []string {
	seen := make(map[string]struct{}, len(reasons))
	out := make([]string, 0, len(reasons))
	for _, r := range reasons {
		if r == "" {
			continue
		}
		if _, ok := seen[r]; ok {
			continue
		}
		seen[r] = struct{}{}
		out = append(out, r)
	}
	return out
}
