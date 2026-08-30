package recommendation

import (
	"math"
	"strings"
	"time"
)

// Score computes a deterministic 0–100 match score and factor breakdown.
func Score(student StudentContext, candidate Candidate, now time.Time) (int, []Factor) {
	if student.ProfileComplete {
		return scoreWithProfile(student, candidate, now)
	}
	return scoreColdStart(candidate, now)
}

func scoreWithProfile(student StudentContext, candidate Candidate, now time.Time) (int, []Factor) {
	w := Weights
	factors := make([]Factor, 0, 8)
	total := 0.0

	careerPts := scoreCareerFamily(student, candidate, float64(w.CareerFamily))
	factors = append(factors, Factor{Key: "career_family", Label: "Career family", Points: careerPts, Max: float64(w.CareerFamily)})
	total += careerPts

	expPts := scoreExperienceLevel(student, candidate, float64(w.ExperienceLevel))
	factors = append(factors, Factor{Key: "experience_level", Label: "Experience level", Points: expPts, Max: float64(w.ExperienceLevel)})
	total += expPts

	skillPts := scoreSkills(student, candidate, float64(w.SkillsOverlap))
	factors = append(factors, Factor{Key: "skills_overlap", Label: "Skills overlap", Points: skillPts, Max: float64(w.SkillsOverlap)})
	total += skillPts

	workPts := scoreWorkArrangement(student, candidate, float64(w.WorkArrangement))
	factors = append(factors, Factor{Key: "work_arrangement", Label: "Work arrangement", Points: workPts, Max: float64(w.WorkArrangement)})
	total += workPts

	locPts := scoreLocation(student, candidate, float64(w.Location))
	factors = append(factors, Factor{Key: "location", Label: "Location preference", Points: locPts, Max: float64(w.Location)})
	total += locPts

	freshPts := scoreFreshness(candidate, now, float64(w.Freshness))
	factors = append(factors, Factor{Key: "freshness", Label: "Recently verified", Points: freshPts, Max: float64(w.Freshness)})
	total += freshPts

	deadlinePts := scoreDeadline(candidate, now, float64(w.DeadlineUrgency))
	factors = append(factors, Factor{Key: "deadline_urgency", Label: "Upcoming deadline", Points: deadlinePts, Max: float64(w.DeadlineUrgency)})
	total += deadlinePts

	if candidate.IsSaved {
		total -= PenaltyAlreadySaved
		factors = append(factors, Factor{Key: "already_saved", Label: "Already saved", Points: -PenaltyAlreadySaved, Max: 0})
	}

	if educationMismatch(student, candidate) {
		total -= PenaltyEducationMismatch
		factors = append(factors, Factor{Key: "education_mismatch", Label: "Education level may not align", Points: -PenaltyEducationMismatch, Max: 0})
	}

	return clampScore(total), factors
}

func scoreColdStart(candidate Candidate, now time.Time) (int, []Factor) {
	w := ColdStartWeights
	factors := make([]Factor, 0, 4)
	total := 35.0 // baseline so incomplete profiles still get useful ordering

	freshPts := scoreFreshness(candidate, now, float64(w.Freshness))
	factors = append(factors, Factor{Key: "freshness", Label: "Recently verified", Points: freshPts, Max: float64(w.Freshness)})
	total += freshPts

	deadlinePts := scoreDeadline(candidate, now, float64(w.DeadlineUrgency))
	factors = append(factors, Factor{Key: "deadline_urgency", Label: "Upcoming deadline", Points: deadlinePts, Max: float64(w.DeadlineUrgency)})
	total += deadlinePts

	expPts := scoreStudentRelevanceBoost(candidate, float64(w.ExperienceLevel))
	factors = append(factors, Factor{Key: "student_relevance", Label: "Student opportunity", Points: expPts, Max: float64(w.ExperienceLevel)})
	total += expPts

	verifyPts := 2.0
	if candidate.VerificationStatus == "verified" {
		factors = append(factors, Factor{Key: "verified_source", Label: "Source verified", Points: verifyPts, Max: 2})
		total += verifyPts
	}

	if candidate.IsSaved {
		total -= PenaltyAlreadySaved
	}

	return clampScore(total), factors
}

func scoreCareerFamily(student StudentContext, candidate Candidate, max float64) float64 {
	if len(student.InferredFamilies) == 0 {
		return 0
	}
	family := derefString(candidate.CareerFamily)
	if family == "" || family == "unknown" {
		return max * 0.15
	}
	for _, want := range student.InferredFamilies {
		if want == family {
			return max
		}
	}
	return 0
}

func scoreExperienceLevel(student StudentContext, candidate Candidate, max float64) float64 {
	exp := derefString(candidate.ExperienceLevel)
	if exp == "" || exp == "unknown" {
		return max * 0.25
	}
	if len(student.PreferredExpLevels) == 0 {
		return scoreStudentRelevanceBoost(candidate, max)
	}
	for _, want := range student.PreferredExpLevels {
		if want == exp {
			return max
		}
	}
	// Partial credit for adjacent student levels.
	if exp == "internship" || exp == "new_grad" || exp == "co_op" || exp == "early_career" {
		return max * 0.5
	}
	return 0
}

func scoreSkills(student StudentContext, candidate Candidate, max float64) float64 {
	if len(student.Skills) == 0 {
		return 0
	}
	if len(candidate.Skills) == 0 {
		return 0
	}
	overlap := countOverlap(student.Skills, candidate.Skills)
	if overlap == 0 {
		// Also check title/tags for skill mentions.
		titleTags := strings.ToLower(candidate.Title + " " + strings.Join(candidate.Tags, " "))
		for _, skill := range student.Skills {
			if strings.Contains(titleTags, skill) {
				overlap++
			}
		}
	}
	if overlap == 0 {
		return 0
	}
	ratio := math.Min(1, float64(overlap)/math.Max(1, float64(len(student.Skills))))
	return max * ratio
}

func scoreWorkArrangement(student StudentContext, candidate Candidate, max float64) float64 {
	pref := strings.ToLower(student.WorkArrangement)
	if pref == "" || pref == "flexible" {
		return max * 0.5
	}
	if strings.EqualFold(pref, candidate.WorkArrangement) {
		return max
	}
	if pref == "hybrid" && candidate.WorkArrangement == "remote" {
		return max * 0.6
	}
	if pref == "remote" && candidate.WorkArrangement == "hybrid" {
		return max * 0.6
	}
	return 0
}

func scoreLocation(student StudentContext, candidate Candidate, max float64) float64 {
	if len(student.PreferredLocations) == 0 {
		return 0
	}
	loc := ""
	if candidate.Location != nil {
		loc = strings.ToLower(*candidate.Location)
	}
	if loc == "" {
		return max * 0.2
	}
	for _, pref := range student.PreferredLocations {
		pref = strings.ToLower(strings.TrimSpace(pref))
		if pref == "" {
			continue
		}
		if strings.Contains(loc, pref) || strings.Contains(pref, loc) {
			return max
		}
		if pref == "remote" && candidate.WorkArrangement == "remote" {
			return max
		}
	}
	return 0
}

func scoreFreshness(candidate Candidate, now time.Time, max float64) float64 {
	if candidate.LastCheckedAt == nil {
		return max * 0.3
	}
	age := now.Sub(*candidate.LastCheckedAt)
	switch {
	case age <= 7*24*time.Hour:
		return max
	case age <= 30*24*time.Hour:
		return max * 0.7
	case age <= 90*24*time.Hour:
		return max * 0.4
	default:
		return max * 0.2
	}
}

func scoreDeadline(candidate Candidate, now time.Time, max float64) float64 {
	if candidate.Deadline == nil {
		return max * 0.3
	}
	days := candidate.Deadline.Sub(now).Hours() / 24
	switch {
	case days <= 7:
		return max
	case days <= 30:
		return max * 0.7
	case days <= 90:
		return max * 0.4
	default:
		return max * 0.2
	}
}

func scoreStudentRelevanceBoost(candidate Candidate, max float64) float64 {
	exp := derefString(candidate.ExperienceLevel)
	switch exp {
	case "internship", "new_grad", "co_op", "early_career", "apprenticeship", "fellowship":
		return max
	case "unknown", "":
		return max * 0.4
	default:
		return max * 0.2
	}
}

func educationMismatch(student StudentContext, candidate Candidate) bool {
	edu := derefString(candidate.EducationLevel)
	if edu != "phd" {
		return false
	}
	// Only flag when we have undergrad signal and no graduate research interest.
	joined := strings.ToLower(strings.Join(append(student.CareerInterests, student.DesiredRoles...), " "))
	if strings.Contains(joined, "phd") || strings.Contains(joined, "doctoral") || strings.Contains(joined, "research") {
		return false
	}
	if student.GraduationYear != nil {
		return true
	}
	return false
}

func countOverlap(a, b []string) int {
	set := make(map[string]struct{}, len(b))
	for _, item := range b {
		set[strings.ToLower(strings.TrimSpace(item))] = struct{}{}
	}
	count := 0
	for _, item := range a {
		if _, ok := set[item]; ok {
			count++
		}
	}
	return count
}

func clampScore(v float64) int {
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	return int(math.Round(v))
}
