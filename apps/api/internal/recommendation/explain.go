package recommendation

import (
	"fmt"
	"strings"
)

// BuildReasons converts scoring factors into human-readable explanations.
func BuildReasons(student StudentContext, candidate Candidate, factors []Factor) (reasons []string, summary string) {
	reasons = make([]string, 0, 6)
	chips := make([]string, 0, 4)

	for _, f := range factors {
		if f.Points <= 0 {
			continue
		}
		switch f.Key {
		case "career_family":
			family := humanFamily(derefString(candidate.CareerFamily))
			if family != "" {
				reasons = append(reasons, fmt.Sprintf("%s matches your career interest", family))
				chips = append(chips, family)
			}
		case "experience_level":
			exp := humanExperience(derefString(candidate.ExperienceLevel))
			if exp != "" {
				reasons = append(reasons, fmt.Sprintf("%s matches your experience level", exp))
				chips = append(chips, exp)
			}
		case "skills_overlap":
			if skills := overlappingSkillLabels(student, candidate); len(skills) > 0 {
				label := strings.Join(skills, ", ")
				reasons = append(reasons, fmt.Sprintf("%s overlaps with your skills", label))
				chips = append(chips, skills[0])
			}
		case "work_arrangement":
			reasons = append(reasons, fmt.Sprintf("%s matches your work preference", humanWorkArrangement(candidate.WorkArrangement)))
		case "location":
			if candidate.Location != nil && *candidate.Location != "" {
				reasons = append(reasons, fmt.Sprintf("Location aligns with your preferences (%s)", *candidate.Location))
			}
		case "freshness":
			reasons = append(reasons, "Recently verified from source")
		case "deadline_urgency":
			if candidate.Deadline != nil {
				reasons = append(reasons, "Deadline is coming up soon")
			}
		case "student_relevance":
			reasons = append(reasons, "Student-relevant opportunity")
		case "verified_source":
			reasons = append(reasons, "Source-verified posting")
		}
	}

	if len(reasons) == 0 {
		if student.ProfileComplete {
			reasons = append(reasons, "Eligible technical opportunity")
		} else {
			reasons = append(reasons, "Popular verified opportunity — complete your profile for better matches")
		}
	}

	if len(chips) > 3 {
		chips = chips[:3]
	}
	summary = strings.Join(chips, " · ")
	return reasons, summary
}

func overlappingSkillLabels(student StudentContext, candidate Candidate) []string {
	if len(student.Skills) == 0 {
		return nil
	}
	candidateSkills := make(map[string]string, len(candidate.Skills))
	for _, s := range candidate.Skills {
		candidateSkills[strings.ToLower(strings.TrimSpace(s))] = s
	}
	var out []string
	titleLower := strings.ToLower(candidate.Title)
	for _, skill := range student.Skills {
		if orig, ok := candidateSkills[skill]; ok {
			out = append(out, orig)
			continue
		}
		if strings.Contains(titleLower, skill) {
			out = append(out, titleCase(skill))
		}
	}
	return out
}

func humanFamily(family string) string {
	switch family {
	case "software_engineering":
		return "Software Engineering"
	case "data_science":
		return "Data Science"
	case "machine_learning_ai":
		return "Machine Learning / AI"
	case "cybersecurity":
		return "Cybersecurity"
	case "product_management_technical":
		return "Technical Product Management"
	case "cloud_infrastructure_devops":
		return "Cloud / DevOps"
	case "quantitative_technology":
		return "Quantitative Technology"
	case "technical_research":
		return "Technical Research"
	case "other_technical":
		return "Technical Role"
	default:
		return ""
	}
}

func humanExperience(level string) string {
	switch level {
	case "internship":
		return "Internship"
	case "co_op":
		return "Co-op"
	case "new_grad":
		return "New Grad"
	case "early_career":
		return "Early Career"
	case "apprenticeship":
		return "Apprenticeship"
	case "fellowship":
		return "Fellowship"
	default:
		return ""
	}
}

func humanWorkArrangement(v string) string {
	switch v {
	case "remote":
		return "Remote"
	case "hybrid":
		return "Hybrid"
	case "on_site":
		return "On-site"
	default:
		return v
	}
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
