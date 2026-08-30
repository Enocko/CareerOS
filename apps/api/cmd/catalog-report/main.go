package main

import (
	"context"
	"fmt"
	"log"

	"github.com/careeros/api/internal/catalogreport"
	"github.com/careeros/api/internal/config"
	"github.com/careeros/api/internal/db"
)

func main() {
	cfg, err := config.LoadIngest()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	repo := catalogreport.NewRepository(pool)

	metrics, err := repo.CollectMetrics(ctx)
	if err != nil {
		log.Fatalf("metrics: %v", err)
	}

	fmt.Println("=== CareerOS Catalog Health Report ===")
	fmt.Println("\n-- Visible catalog --")
	fmt.Printf("Employment (verified, open): %d\n", metrics.EmploymentVisible)
	fmt.Printf("Research candidate programs (verified, open): %d\n", metrics.ResearchCandidate)
	fmt.Printf("  applications open: %d\n", metrics.ResearchOpen)
	fmt.Printf("  applications upcoming: %d\n", metrics.ResearchUpcoming)
	fmt.Printf("  application status unknown: %d\n", metrics.ResearchUnknown)
	fmt.Printf("  applications closed: %d\n", metrics.ResearchClosed)
	fmt.Printf("Verified open listings (all types): %d\n", metrics.VerifiedListings)
	fmt.Printf("Stale listings: %d\n", metrics.StaleListings)
	fmt.Printf("Closed listings (historical): %d\n", metrics.ClosedListings)
	fmt.Printf("With deadline: %d\n", metrics.WithDeadline)
	fmt.Printf("Without deadline: %d\n", metrics.WithoutDeadline)
	fmt.Printf("Checked within 7 days: %d\n", metrics.CheckedWithin7Days)
	fmt.Printf("Pending student reports: %d\n", metrics.PendingReports)

	fmt.Println("\n-- Employment by provider --")
	for name, count := range metrics.ProviderCounts {
		fmt.Printf("  %s: %d\n", name, count)
	}

	groups, totalGroups, excessRecords, err := repo.FindDuplicateGroups(ctx, 15)
	if err != nil {
		log.Fatalf("duplicates: %v", err)
	}
	fmt.Println("\n-- Cross-source duplicate audit (deterministic) --")
	fmt.Printf("Candidate duplicate groups: %d\n", totalGroups)
	fmt.Printf("Likely excess duplicate records: %d\n", excessRecords)
	if metrics.EmploymentVisible > 0 {
		rate := float64(excessRecords) / float64(metrics.EmploymentVisible) * 100
		fmt.Printf("Estimated duplicate rate: %.2f%%\n", rate)
	}
	if len(groups) > 0 {
		fmt.Println("Sample groups:")
		for _, g := range groups {
			fmt.Printf("  %s — %s (%d records)\n", g.Organization, g.Title, g.Count)
		}
	}

	health, err := repo.ListSourceHealth(ctx)
	if err != nil {
		log.Fatalf("source health: %v", err)
	}
	fmt.Println("\n-- Source health --")
	for _, row := range health {
		status := "unknown"
		if row.LastRunStatus != nil {
			status = *row.LastRunStatus
		}
		lastOK := "never"
		if row.LastSuccessAt != nil {
			lastOK = row.LastSuccessAt.Format("2006-01-02 15:04 UTC")
		}
		fmt.Printf("  %s (%s) enabled=%v last_success=%s last_run=%s verified_open=%d consecutive_fails=%d",
			row.SourceName, row.Adapter, row.Enabled, lastOK, status, row.VerifiedOpenCount, row.ConsecutiveFails)
		if row.LastErrorCode != nil && *row.LastErrorCode != "" {
			fmt.Printf(" error=%s", *row.LastErrorCode)
		}
		fmt.Println()
	}

	urlIssues, err := repo.AuditApplicationURLs(ctx, 10)
	if err != nil {
		log.Fatalf("url audit: %v", err)
	}
	fmt.Println("\n-- Application URL audit (sampled) --")
	if len(urlIssues) == 0 {
		fmt.Println("  No obvious URL issues in sample.")
	} else {
		for _, issue := range urlIssues {
			fmt.Printf("  [%s] %s — %s (%s)\n", issue.Source, issue.Title, issue.Issue, issue.ApplicationURL)
		}
	}
}
