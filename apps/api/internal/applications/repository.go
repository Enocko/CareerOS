package applications

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles application persistence.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new applications Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create inserts a new application with initial status history.
func (r *Repository) Create(ctx context.Context, studentID uuid.UUID, req CreateRequest) (*Application, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var oppStatus, oppType string
	err = tx.QueryRow(ctx, `
		SELECT status, opportunity_type FROM opportunities WHERE id = $1
	`, req.OpportunityID).Scan(&oppStatus, &oppType)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOpportunityNotFound
		}
		return nil, fmt.Errorf("check opportunity: %w", err)
	}
	if oppStatus != "open" {
		return nil, ErrOpportunityClosed
	}
	if oppType != "employment" {
		return nil, ErrOpportunityNotEmployment
	}

	var app Application
	err = tx.QueryRow(ctx, `
		INSERT INTO applications (student_id, opportunity_id, current_status, notes)
		VALUES ($1, $2, 'saved', $3)
		RETURNING id, opportunity_id, current_status, date_applied, notes,
		          next_action, next_action_date, interview_date, created_at, updated_at
	`, studentID, req.OpportunityID, req.Notes).Scan(
		&app.ID, &app.OpportunityID, &app.CurrentStatus, &app.DateApplied, &app.Notes,
		&app.NextAction, &app.NextActionDate, &app.InterviewDate, &app.CreatedAt, &app.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("insert application: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO application_status_history (application_id, from_status, to_status, changed_by)
		VALUES ($1, NULL, 'saved', $2)
	`, app.ID, studentID)
	if err != nil {
		return nil, fmt.Errorf("insert status history: %w", err)
	}

	opp, err := r.getOpportunityBrief(ctx, tx, req.OpportunityID, false)
	if err != nil {
		return nil, err
	}
	app.Opportunity = opp

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &app, nil
}

// GetByID returns an application with status history for the owning student.
func (r *Repository) GetByID(ctx context.Context, studentID, applicationID uuid.UUID) (*Application, error) {
	app, err := r.getApplicationTx(ctx, r.pool, studentID, applicationID)
	if err != nil {
		return nil, err
	}

	opp, err := r.getOpportunityBrief(ctx, r.pool, app.OpportunityID, true)
	if err != nil {
		return nil, err
	}
	app.Opportunity = opp

	rows, err := r.pool.Query(ctx, `
		SELECT id, from_status, to_status, changed_at
		FROM application_status_history
		WHERE application_id = $1
		ORDER BY changed_at ASC
	`, applicationID)
	if err != nil {
		return nil, fmt.Errorf("get status history: %w", err)
	}
	defer rows.Close()

	var history []StatusHistory
	for rows.Next() {
		var h StatusHistory
		if err := rows.Scan(&h.ID, &h.FromStatus, &h.ToStatus, &h.ChangedAt); err != nil {
			return nil, fmt.Errorf("scan history: %w", err)
		}
		history = append(history, h)
	}
	if history == nil {
		history = []StatusHistory{}
	}
	app.StatusHistory = history

	return app, rows.Err()
}

// List returns paginated applications for a student.
func (r *Repository) List(ctx context.Context, studentID uuid.UUID, filter ListFilter) ([]Application, int, error) {
	conditions := []string{"a.student_id = $1"}
	args := []any{studentID}
	argIndex := 2

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("a.current_status = $%d", argIndex))
		args = append(args, filter.Status)
		argIndex++
	}

	where := fmt.Sprintf("WHERE %s", joinConditions(conditions))

	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM applications a %s`, where)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count applications: %w", err)
	}

	offset := (filter.Page - 1) * filter.PerPage
	limitArg := fmt.Sprintf("$%d", argIndex)
	offsetArg := fmt.Sprintf("$%d", argIndex+1)
	args = append(args, filter.PerPage, offset)

	listQuery := fmt.Sprintf(`
		SELECT a.id, a.opportunity_id, a.current_status, a.date_applied, a.notes,
		       a.next_action, a.next_action_date, a.interview_date, a.created_at, a.updated_at,
		       o.id, o.title, o.organization_name, o.category, o.deadline, o.application_url
		FROM applications a
		JOIN opportunities o ON o.id = a.opportunity_id
		%s
		ORDER BY a.updated_at DESC
		LIMIT %s OFFSET %s
	`, where, limitArg, offsetArg)

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list applications: %w", err)
	}
	defer rows.Close()

	var results []Application
	for rows.Next() {
		var app Application
		var opp OpportunityBrief
		if err := rows.Scan(
			&app.ID, &app.OpportunityID, &app.CurrentStatus, &app.DateApplied, &app.Notes,
			&app.NextAction, &app.NextActionDate, &app.InterviewDate, &app.CreatedAt, &app.UpdatedAt,
			&opp.ID, &opp.Title, &opp.OrganizationName, &opp.Category, &opp.Deadline, &opp.ApplicationURL,
		); err != nil {
			return nil, 0, fmt.Errorf("scan application: %w", err)
		}
		app.Opportunity = &opp
		results = append(results, app)
	}

	if results == nil {
		results = []Application{}
	}

	return results, total, rows.Err()
}

// Update modifies an application and records status history if status changed.
func (r *Repository) Update(ctx context.Context, studentID, applicationID uuid.UUID, req UpdateRequest) (*Application, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	current, err := r.getApplicationTx(ctx, tx, studentID, applicationID)
	if err != nil {
		return nil, err
	}

	newStatus := current.CurrentStatus
	if req.CurrentStatus != nil {
		newStatus = *req.CurrentStatus
	}

	dateApplied := current.DateApplied
	if newStatus == "applied" && dateApplied == nil {
		now := time.Now().UTC().Truncate(24 * time.Hour)
		dateApplied = &now
	}

	notes := coalesceStringPtr(req.Notes, current.Notes)
	nextAction := coalesceStringPtr(req.NextAction, current.NextAction)
	nextActionDate := coalesceTimePtr(req.NextActionDate, current.NextActionDate)
	interviewDate := coalesceTimePtr(req.InterviewDate, current.InterviewDate)

	var updated Application
	err = tx.QueryRow(ctx, `
		UPDATE applications
		SET current_status = $1, date_applied = $2, notes = $3,
		    next_action = $4, next_action_date = $5, interview_date = $6,
		    updated_at = now()
		WHERE id = $7 AND student_id = $8
		RETURNING id, opportunity_id, current_status, date_applied, notes,
		          next_action, next_action_date, interview_date, created_at, updated_at
	`, newStatus, dateApplied, notes, nextAction, nextActionDate, interviewDate, applicationID, studentID).Scan(
		&updated.ID, &updated.OpportunityID, &updated.CurrentStatus, &updated.DateApplied, &updated.Notes,
		&updated.NextAction, &updated.NextActionDate, &updated.InterviewDate, &updated.CreatedAt, &updated.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update application: %w", err)
	}

	if req.CurrentStatus != nil && *req.CurrentStatus != current.CurrentStatus {
		fromStatus := current.CurrentStatus
		_, err = tx.Exec(ctx, `
			INSERT INTO application_status_history (application_id, from_status, to_status, changed_by)
			VALUES ($1, $2, $3, $4)
		`, applicationID, fromStatus, newStatus, studentID)
		if err != nil {
			return nil, fmt.Errorf("insert status history: %w", err)
		}
	}

	opp, err := r.getOpportunityBrief(ctx, tx, updated.OpportunityID, true)
	if err != nil {
		return nil, err
	}
	updated.Opportunity = opp

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &updated, nil
}

// Delete removes a tracking record that has not progressed past preparation.
func (r *Repository) Delete(ctx context.Context, studentID, applicationID uuid.UUID) error {
	current, err := r.getApplicationTx(ctx, r.pool, studentID, applicationID)
	if err != nil {
		return err
	}
	if !RemovableStatuses[current.CurrentStatus] {
		return ErrCannotRemove
	}

	tag, err := r.pool.Exec(ctx, `
		DELETE FROM applications
		WHERE id = $1 AND student_id = $2
	`, applicationID, studentID)
	if err != nil {
		return fmt.Errorf("delete application: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) getApplicationTx(ctx context.Context, q pgxQuerier, studentID, applicationID uuid.UUID) (*Application, error) {
	var app Application
	err := q.QueryRow(ctx, `
		SELECT id, opportunity_id, current_status, date_applied, notes,
		       next_action, next_action_date, interview_date, created_at, updated_at
		FROM applications
		WHERE id = $1 AND student_id = $2
	`, applicationID, studentID).Scan(
		&app.ID, &app.OpportunityID, &app.CurrentStatus, &app.DateApplied, &app.Notes,
		&app.NextAction, &app.NextActionDate, &app.InterviewDate, &app.CreatedAt, &app.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get application: %w", err)
	}
	return &app, nil
}

func (r *Repository) getOpportunityBrief(ctx context.Context, q pgxQuerier, opportunityID uuid.UUID, includeAppURL bool) (*OpportunityBrief, error) {
	var opp OpportunityBrief
	var query string
	if includeAppURL {
		query = `SELECT id, title, organization_name, category, deadline, application_url FROM opportunities WHERE id = $1`
		err := q.QueryRow(ctx, query, opportunityID).Scan(
			&opp.ID, &opp.Title, &opp.OrganizationName, &opp.Category, &opp.Deadline, &opp.ApplicationURL,
		)
		if err != nil {
			return nil, fmt.Errorf("get opportunity brief: %w", err)
		}
	} else {
		query = `SELECT id, title, organization_name, category, deadline FROM opportunities WHERE id = $1`
		err := q.QueryRow(ctx, query, opportunityID).Scan(
			&opp.ID, &opp.Title, &opp.OrganizationName, &opp.Category, &opp.Deadline,
		)
		if err != nil {
			return nil, fmt.Errorf("get opportunity brief: %w", err)
		}
	}
	return &opp, nil
}

type pgxQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func joinConditions(conditions []string) string {
	result := ""
	for i, c := range conditions {
		if i > 0 {
			result += " AND "
		}
		result += c
	}
	return result
}

func coalesceStringPtr(newVal, current *string) *string {
	if newVal != nil {
		return newVal
	}
	return current
}

func coalesceTimePtr(newVal, current *time.Time) *time.Time {
	if newVal != nil {
		return newVal
	}
	return current
}
