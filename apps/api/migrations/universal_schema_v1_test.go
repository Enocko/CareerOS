package opportunitytype_test

import (
	"context"
	"os"
	"testing"

	"github.com/careeros/api/internal/db"
	"github.com/careeros/api/internal/opportunitytype"
)

func TestUniversalSchemaV1MigrationData(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://careeros:careeros@localhost:5433/careeros?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		t.Skipf("database not available: %v", err)
	}
	defer pool.Close()

	var hasColumn bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'opportunities' AND column_name = 'opportunity_type'
		)
	`).Scan(&hasColumn)
	if err != nil {
		t.Fatalf("check column: %v", err)
	}
	if !hasColumn {
		t.Skip("migration 000010 not applied; run make migrate")
	}

	var total, employment, unmapped int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM opportunities`).Scan(&total)
	if err != nil {
		t.Fatalf("count total: %v", err)
	}
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM opportunities WHERE opportunity_type = 'employment'
	`).Scan(&employment)
	if err != nil {
		t.Fatalf("count employment: %v", err)
	}
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM opportunities
		WHERE opportunity_type IS NULL OR opportunity_type = ''
	`).Scan(&unmapped)
	if err != nil {
		t.Fatalf("count unmapped: %v", err)
	}

	t.Logf("migration data: total=%d employment=%d unmapped=%d", total, employment, unmapped)

	if unmapped != 0 {
		t.Fatalf("expected unmapped opportunity_type count 0, got %d", unmapped)
	}

	rows, err := pool.Query(ctx, `
		SELECT opportunity_type, COUNT(*) FROM opportunities GROUP BY opportunity_type ORDER BY opportunity_type
	`)
	if err != nil {
		t.Fatalf("distribution: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var typ string
		var count int
		if err := rows.Scan(&typ, &count); err != nil {
			t.Fatal(err)
		}
		if !opportunitytype.ValidType(typ) {
			t.Fatalf("invalid opportunity_type in database: %q", typ)
		}
		t.Logf("opportunity_type=%s count=%d", typ, count)
	}

	var fellowshipExp int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM opportunities WHERE experience_level = 'fellowship'
	`).Scan(&fellowshipExp)
	if err != nil {
		t.Fatalf("count fellowship experience_level: %v", err)
	}
	if fellowshipExp != 0 {
		t.Fatalf("expected 0 fellowship experience_level rows, got %d", fellowshipExp)
	}
}
