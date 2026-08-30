package recommendation

import (
	"strings"

	"github.com/careeros/api/internal/profile"
)

type familyRule struct {
	family  string
	needles []string
}

var familyRules = []familyRule{
	{family: "software_engineering", needles: []string{"software", "swe", "backend", "frontend", "full stack", "fullstack", "developer", "computer science", "cs "}},
	{family: "data_science", needles: []string{"data science", "data scientist", "data analyst"}},
	{family: "machine_learning_ai", needles: []string{"machine learning", "ml", "artificial intelligence", " ai", "deep learning"}},
	{family: "cybersecurity", needles: []string{"cyber", "security", "infosec"}},
	{family: "product_management_technical", needles: []string{"product management", "technical product", "product manager"}},
	{family: "cloud_infrastructure_devops", needles: []string{"devops", "sre", "cloud", "infrastructure", "platform engineer"}},
	{family: "quantitative_technology", needles: []string{"quant", "quantitative"}},
	{family: "technical_research", needles: []string{"research engineer", "research scientist"}},
}

// StudentContextFromProfile builds scoring context from an existing profile.
// Returns a cold-start context when profile is nil.
func StudentContextFromProfile(p *profile.Profile) StudentContext {
	ctx := StudentContext{}
	if p == nil {
		return ctx
	}

	ctx.HasProfile = true
	if p.Major != nil {
		ctx.Major = strings.TrimSpace(*p.Major)
	}
	ctx.GraduationYear = p.GraduationYear
	ctx.CareerInterests = append([]string{}, p.CareerInterests...)
	ctx.DesiredRoles = append([]string{}, p.DesiredRoles...)
	ctx.Skills = uniqueLower(append(append([]string{}, p.Skills...), p.Technologies...))
	ctx.Technologies = append([]string{}, p.Technologies...)
	ctx.PreferredLocations = append([]string{}, p.PreferredLocations...)
	if p.WorkArrangement != nil {
		ctx.WorkArrangement = strings.TrimSpace(*p.WorkArrangement)
	}
	if p.ExperienceLevel != nil {
		ctx.ExperienceLevel = strings.TrimSpace(*p.ExperienceLevel)
	}

	ctx.InferredFamilies = inferCareerFamilies(ctx)
	ctx.PreferredExpLevels = mapProfileExperience(ctx.ExperienceLevel)
	ctx.ProfileComplete = isProfileComplete(ctx)
	return ctx
}

func inferCareerFamilies(ctx StudentContext) []string {
	texts := make([]string, 0, 4+len(ctx.CareerInterests)+len(ctx.DesiredRoles))
	if ctx.Major != "" {
		texts = append(texts, ctx.Major)
	}
	texts = append(texts, ctx.CareerInterests...)
	texts = append(texts, ctx.DesiredRoles...)

	joined := " " + strings.ToLower(strings.Join(texts, " ")) + " "
	seen := map[string]struct{}{}
	var families []string
	for _, rule := range familyRules {
		for _, needle := range rule.needles {
			if strings.Contains(joined, " "+strings.ToLower(needle)) || strings.Contains(joined, strings.ToLower(needle)) {
				if _, ok := seen[rule.family]; !ok {
					seen[rule.family] = struct{}{}
					families = append(families, rule.family)
				}
				break
			}
		}
	}
	return families
}

func mapProfileExperience(level string) []string {
	switch strings.ToLower(level) {
	case "intern":
		return []string{"internship", "co_op", "fellowship"}
	case "entry":
		return []string{"new_grad", "early_career", "apprenticeship", "internship", "co_op"}
	case "mid", "senior":
		return []string{"early_career", "new_grad"}
	default:
		return nil
	}
}

func isProfileComplete(ctx StudentContext) bool {
	if len(ctx.InferredFamilies) > 0 {
		return true
	}
	if len(ctx.Skills) > 0 {
		return true
	}
	if ctx.WorkArrangement != "" {
		return true
	}
	if len(ctx.PreferredLocations) > 0 {
		return true
	}
	if ctx.ExperienceLevel != "" {
		return true
	}
	if ctx.Major != "" {
		return true
	}
	return false
}

func uniqueLower(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(strings.ToLower(item))
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
