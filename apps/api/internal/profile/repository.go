package profile

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles student profile persistence.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new profile Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetByUserID retrieves a profile for the given user.
func (r *Repository) GetByUserID(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	var p Profile
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, first_name, last_name, university, major, graduation_year,
		       career_interests, desired_roles, skills, technologies, preferred_locations,
		       work_arrangement, experience_level, github_url, linkedin_url, portfolio_url,
		       created_at, updated_at
		FROM student_profiles
		WHERE user_id = $1
	`, userID).Scan(
		&p.ID, &p.UserID, &p.FirstName, &p.LastName, &p.University, &p.Major, &p.GraduationYear,
		&p.CareerInterests, &p.DesiredRoles, &p.Skills, &p.Technologies, &p.PreferredLocations,
		&p.WorkArrangement, &p.ExperienceLevel, &p.GithubURL, &p.LinkedinURL, &p.PortfolioURL,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("get profile: %w", err)
	}

	p.CareerInterests = normalizeStringSlice(p.CareerInterests)
	p.DesiredRoles = normalizeStringSlice(p.DesiredRoles)
	p.Skills = normalizeStringSlice(p.Skills)
	p.Technologies = normalizeStringSlice(p.Technologies)
	p.PreferredLocations = normalizeStringSlice(p.PreferredLocations)

	return &p, nil
}

// Upsert creates or updates a profile for the given user.
func (r *Repository) Upsert(ctx context.Context, userID uuid.UUID, req UpdateRequest) (*Profile, error) {
	university := universityOrDefault(req.University)

	var p Profile
	err := r.pool.QueryRow(ctx, `
		INSERT INTO student_profiles (
			user_id, first_name, last_name, university, major, graduation_year,
			career_interests, desired_roles, skills, technologies, preferred_locations,
			work_arrangement, experience_level, github_url, linkedin_url, portfolio_url
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16
		)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			university = EXCLUDED.university,
			major = EXCLUDED.major,
			graduation_year = EXCLUDED.graduation_year,
			career_interests = EXCLUDED.career_interests,
			desired_roles = EXCLUDED.desired_roles,
			skills = EXCLUDED.skills,
			technologies = EXCLUDED.technologies,
			preferred_locations = EXCLUDED.preferred_locations,
			work_arrangement = EXCLUDED.work_arrangement,
			experience_level = EXCLUDED.experience_level,
			github_url = EXCLUDED.github_url,
			linkedin_url = EXCLUDED.linkedin_url,
			portfolio_url = EXCLUDED.portfolio_url,
			updated_at = now()
		RETURNING id, user_id, first_name, last_name, university, major, graduation_year,
		          career_interests, desired_roles, skills, technologies, preferred_locations,
		          work_arrangement, experience_level, github_url, linkedin_url, portfolio_url,
		          created_at, updated_at
	`,
		userID,
		stringOrNil(req.FirstName),
		stringOrNil(req.LastName),
		university,
		stringOrNil(req.Major),
		req.GraduationYear,
		normalizeStringSlice(req.CareerInterests),
		normalizeStringSlice(req.DesiredRoles),
		normalizeStringSlice(req.Skills),
		normalizeStringSlice(req.Technologies),
		normalizeStringSlice(req.PreferredLocations),
		stringOrNil(req.WorkArrangement),
		stringOrNil(req.ExperienceLevel),
		stringOrNil(req.GithubURL),
		stringOrNil(req.LinkedinURL),
		stringOrNil(req.PortfolioURL),
	).Scan(
		&p.ID, &p.UserID, &p.FirstName, &p.LastName, &p.University, &p.Major, &p.GraduationYear,
		&p.CareerInterests, &p.DesiredRoles, &p.Skills, &p.Technologies, &p.PreferredLocations,
		&p.WorkArrangement, &p.ExperienceLevel, &p.GithubURL, &p.LinkedinURL, &p.PortfolioURL,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert profile: %w", err)
	}

	p.CareerInterests = normalizeStringSlice(p.CareerInterests)
	p.DesiredRoles = normalizeStringSlice(p.DesiredRoles)
	p.Skills = normalizeStringSlice(p.Skills)
	p.Technologies = normalizeStringSlice(p.Technologies)
	p.PreferredLocations = normalizeStringSlice(p.PreferredLocations)

	return &p, nil
}
