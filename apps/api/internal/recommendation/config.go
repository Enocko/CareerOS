package recommendation

// Weights defines the deterministic scoring scale (0–100).
// All profile-match components are centralized here for easy tuning.
var Weights = ScoreWeights{
	CareerFamily:    30,
	ExperienceLevel: 20,
	SkillsOverlap:   20,
	WorkArrangement: 10,
	Location:        10,
	Freshness:       5,
	DeadlineUrgency: 5,
}

// ScoreWeights holds point allocations for each ranking signal.
type ScoreWeights struct {
	CareerFamily    int
	ExperienceLevel int
	SkillsOverlap   int
	WorkArrangement int
	Location        int
	Freshness       int
	DeadlineUrgency int
}

// MaxScore returns the theoretical maximum when all signals match.
func (w ScoreWeights) MaxScore() int {
	return w.CareerFamily + w.ExperienceLevel + w.SkillsOverlap +
		w.WorkArrangement + w.Location + w.Freshness + w.DeadlineUrgency
}

// ColdStartWeights are used when profile signals are sparse.
var ColdStartWeights = ScoreWeights{
	Freshness:       3,
	DeadlineUrgency: 2,
	ExperienceLevel: 5, // internship/new-grad relevance boost
}

// Ranking adjustments (not hard filters).
const (
	PenaltyAlreadySaved      = 3
	PenaltyEducationMismatch = 10
)
