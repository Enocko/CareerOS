package admin

import (
	"net/http"
	"time"

	"github.com/careeros/api/internal/catalogreport"
	"github.com/careeros/api/internal/jobs"
	"github.com/careeros/api/internal/opportunityreports"
	"github.com/careeros/api/internal/platform"
	"github.com/careeros/api/internal/researchverification"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler serves lightweight operator overview endpoints.
type Handler struct {
	catalogRepo *catalogreport.Repository
	reportSvc   *opportunityreports.Service
	researchSvc *researchverification.Service
	jobsRepo    *jobs.Repository
}

// NewHandler creates an admin overview handler.
func NewHandler(
	pool *pgxpool.Pool,
	reportSvc *opportunityreports.Service,
	researchSvc *researchverification.Service,
) *Handler {
	return &Handler{
		catalogRepo: catalogreport.NewRepository(pool),
		reportSvc:   reportSvc,
		researchSvc: researchSvc,
		jobsRepo:    jobs.NewRepository(pool),
	}
}

// Overview is the operator landing summary.
type Overview struct {
	CatalogMetrics      *catalogreport.Metrics          `json:"catalog_metrics"`
	PendingReports      int                             `json:"pending_reports"`
	ResearchMetrics     researchverification.Metrics    `json:"research_metrics"`
	JobMetrics          jobs.Metrics                    `json:"job_metrics"`
	SourceHealth        []catalogreport.SourceHealthRow `json:"source_health"`
	DuplicateGroups     int                             `json:"duplicate_groups"`
	ExcessDuplicateRecs int                             `json:"excess_duplicate_records"`
}

// GetOverview handles GET /api/v1/admin/overview
func (h *Handler) GetOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now().UTC()

	metrics, err := h.catalogRepo.CollectMetrics(ctx)
	if err != nil {
		platform.WriteError(w, platform.InternalError())
		return
	}
	pending, err := h.reportSvc.CountPending(ctx)
	if err != nil {
		platform.WriteError(w, platform.InternalError())
		return
	}
	researchMetrics, err := h.researchSvc.GetMetrics(ctx)
	if err != nil {
		platform.WriteError(w, platform.InternalError())
		return
	}
	jobMetrics, err := h.jobsRepo.Metrics(ctx, now)
	if err != nil {
		platform.WriteError(w, platform.InternalError())
		return
	}
	sourceHealth, err := h.catalogRepo.ListSourceHealth(ctx)
	if err != nil {
		platform.WriteError(w, platform.InternalError())
		return
	}
	_, totalGroups, excessRecords, err := h.catalogRepo.FindDuplicateGroups(ctx, 1)
	if err != nil {
		platform.WriteError(w, platform.InternalError())
		return
	}

	platform.WriteJSON(w, http.StatusOK, Overview{
		CatalogMetrics:      metrics,
		PendingReports:      pending,
		ResearchMetrics:     researchMetrics,
		JobMetrics:          *jobMetrics,
		SourceHealth:        sourceHealth,
		DuplicateGroups:     totalGroups,
		ExcessDuplicateRecs: excessRecords,
	})
}
