package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/careeros/api/internal/db"
	"github.com/careeros/api/internal/ingestion"
	"github.com/careeros/api/internal/ingestion/ashby"
	"github.com/careeros/api/internal/ingestion/greenhouse"
	"github.com/careeros/api/internal/ingestion/lever"
	"github.com/careeros/api/internal/ingestion/relevance"
	v2 "github.com/careeros/api/internal/ingestion/relevance/v2"
)

func main() {
	ctx := context.Background()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://careeros:careeros@localhost:5433/careeros?sslmode=disable"
	}

	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("database connect: %v", err)
	}
	defer pool.Close()

	repo := ingestion.NewRepository(pool)
	httpClient := &http.Client{Timeout: 45 * time.Second}

	fmt.Println("=== Relevance Engine v2: Reclassify Catalog ===")
	rows, err := repo.ListVerifiedOpportunities(ctx)
	if err != nil {
		log.Fatalf("list opportunities: %v", err)
	}
	fmt.Printf("Evaluating %d verified open opportunities\n\n", len(rows))

	for _, row := range rows {
		c := v2.Classify(row.Title, row.Description)
		if err := repo.UpdateClassification(ctx, row.ID,
			string(c.ExperienceLevel), string(c.CareerFamily),
			string(c.EducationLevel), string(c.RelevanceTier), c.Reasons); err != nil {
			log.Fatalf("update classification %s: %v", row.ID, err)
		}
	}

	stats, err := repo.CountClassificationStats(ctx)
	if err != nil {
		log.Fatalf("stats: %v", err)
	}
	printStats(stats)

	fmt.Println("\n=== False Negative Audit (upstream filtered titles) ===")
	auditFalseNegatives(ctx, repo, httpClient)

	fmt.Println("\n=== Manual Audit Samples ===")
	printManualSamples(ctx, repo)
}

func printStats(stats *ingestion.ClassificationStats) {
	fmt.Printf("Total verified: %d\n", stats.Total)
	fmt.Printf("Technical feed: %d\n", stats.TechnicalFeed)
	fmt.Printf("Non-technical: %d\n", stats.NonTechnical)
	fmt.Printf("Ambiguous: %d\n", stats.Ambiguous)
	fmt.Println("\nCareer family distribution:")
	printSortedMap(stats.ByCareerFamily)
	fmt.Println("\nExperience level distribution:")
	printSortedMap(stats.ByExperienceLevel)
	fmt.Println("\nEducation level distribution:")
	printSortedMap(stats.ByEducationLevel)
}

func printSortedMap(m map[string]int) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %s: %d\n", k, m[k])
	}
}

var falseNegativePatterns = []string{
	"software engineer i",
	"associate software engineer",
	"graduate software engineer",
	"university graduate",
	"entry level engineer",
	"engineering co-op",
	"technology analyst",
	"developer intern",
	"research engineer intern",
	"applied scientist intern",
}

func auditFalseNegatives(ctx context.Context, repo *ingestion.Repository, client *http.Client) {
	sources, err := repo.ListEnabledSources(ctx)
	if err != nil {
		log.Printf("list sources: %v", err)
		return
	}

	gh := greenhouse.NewAdapter(client)
	ab := ashby.NewAdapter(client)
	lv := lever.NewAdapter(client)

	type miss struct {
		source string
		adapter string
		title  string
		reason string
	}
	var v1Misses []miss
	var v2Recovered []miss

	for _, source := range sources {
		var titles []string
		switch source.Adapter {
		case "greenhouse":
			var cfg greenhouse.Config
			if err := json.Unmarshal(source.Config, &cfg); err != nil {
				continue
			}
			titles, err = gh.ListAllTitles(ctx, cfg)
		case "ashby":
			var cfg ashby.Config
			if err := json.Unmarshal(source.Config, &cfg); err != nil {
				continue
			}
			titles, err = ab.ListAllTitles(ctx, cfg)
		case "lever":
			var cfg lever.Config
			if err := json.Unmarshal(source.Config, &cfg); err != nil {
				continue
			}
			titles, err = lv.ListAllTitles(ctx, cfg)
		default:
			continue
		}
		if err != nil {
			continue
		}

		for _, title := range titles {
			lower := strings.ToLower(title)
			matchesPattern := false
			for _, p := range falseNegativePatterns {
				if strings.Contains(lower, p) {
					matchesPattern = true
					break
				}
			}
			if !matchesPattern {
				continue
			}

			v1 := relevance.ClassifyStudentRelevance(title)
			c := v2.Classify(title, "")
			v2Persist := v2.ShouldPersistSourceRecord(title, c)

			if !v1.Relevant && v2Persist {
				v2Recovered = append(v2Recovered, miss{source.Name, source.Adapter, title, c.PrimaryReasonCode()})
			}
			if !v1.Relevant && !v2Persist {
				v1Misses = append(v1Misses, miss{source.Name, source.Adapter, title, "still_filtered"})
			}
		}
	}

	fmt.Printf("v2 recovered from v1 filter (confirmed improvements): %d\n", len(v2Recovered))
	for _, m := range v2Recovered {
		fmt.Printf("  [%s/%s] %s → %s\n", m.adapter, m.source, m.title, m.reason)
	}

	fmt.Printf("\nPattern matches still filtered by v2 (review for false negatives): %d\n", len(v1Misses))
	for _, m := range v1Misses {
		fmt.Printf("  [%s/%s] %s\n", m.adapter, m.source, m.title)
	}
}

func printManualSamples(ctx context.Context, repo *ingestion.Repository) {
	queries := []struct {
		label string
		tier  string
	}{
		{"retained technical", "high_confidence_technical"},
		{"non-technical", "high_confidence_non_technical"},
		{"ambiguous", "ambiguous"},
	}
	for _, q := range queries {
		fmt.Printf("\n--- %s (up to 30) ---\n", q.label)
		samples, err := repo.SampleByRelevanceTier(ctx, q.tier, 30)
		if err != nil {
			fmt.Printf("  error: %v\n", err)
			continue
		}
		for _, row := range samples {
			fmt.Printf("  %s | %s | %s\n", row.Title, deref(row.CareerFamily), deref(row.RelevanceTier))
		}
	}
}

func deref(s *string) string {
	if s == nil {
		return "unset"
	}
	return *s
}
