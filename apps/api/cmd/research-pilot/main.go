package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/careeros/api/internal/db"
	"github.com/careeros/api/internal/researchverification"
	"github.com/google/uuid"
)

type pilotEntry struct {
	OpportunityID         string
	ApplicationStatus     string
	VerificationSourceURL string
	ApplicationURL        string
	Deadline              string
	OpensAt               string
	CycleLabel            string
	Notes                 string
}

// Pilot verifications researched from official program pages (Aug 2026 context).
var pilotEntries = []pilotEntry{
	{
		OpportunityID:         "404eb472-0110-418f-8577-7d17093f86a6",
		ApplicationStatus:     "closed",
		VerificationSourceURL: "https://www.jmu.edu/mathstat/reu/index.shtml",
		Deadline:              "2026-02-20",
		CycleLabel:            "Summer 2026",
		Notes:                 "Official page states application deadline February 20, 2026 via NSF ETAP.",
	},
	{
		OpportunityID:         "a749131e-0e50-440e-9723-72b04715f4a4",
		ApplicationStatus:     "unknown",
		VerificationSourceURL: "",
		Notes:                 "Program URL in abstract appears malformed; availability not verified.",
	},
	{
		OpportunityID:         "c01055f3-4074-4f2a-86b5-9ea864d916c1",
		ApplicationStatus:     "unknown",
		VerificationSourceURL: "https://www.cpp.edu/camp-reu/index.shtml",
		CycleLabel:            "Summer 2026",
		Notes:                 "Official page describes program but no current application deadline published.",
	},
	{
		OpportunityID:         "9c709192-147f-4fb0-ad05-83e633b4fb0c",
		ApplicationStatus:     "closed",
		VerificationSourceURL: "https://www.unlv.edu/lifesciences/moereu",
		Deadline:              "2026-03-13",
		CycleLabel:            "Summer 2026",
		Notes:                 "Official page states application deadline March 13, 2026 11:59 PM ET.",
	},
	{
		OpportunityID:         "2c895a5c-e59b-4f92-901f-e0e5e9c328f5",
		ApplicationStatus:     "upcoming",
		VerificationSourceURL: "https://www.calacademy.org/summer-systematics-institute",
		OpensAt:               "2026-12-01",
		CycleLabel:            "Summer 2027",
		Notes:                 "2026 applications closed; official page indicates 2027 applications expected around December 2026.",
	},
	{
		OpportunityID:         "1ea75c53-bc15-4a25-b71a-280db5f00b20",
		ApplicationStatus:     "closed",
		VerificationSourceURL: "https://www.danforthcenter.org/our-work/education-outreach/undergraduate-program/internship-program/",
		CycleLabel:            "Summer 2026",
		Notes:                 "Official page states applications for the 2026 program are closed.",
	},
	{
		OpportunityID:         "e0345e26-a88e-46b0-a09a-f55eed9bbea3",
		ApplicationStatus:     "unknown",
		VerificationSourceURL: "https://cws.auburn.edu/apspi/pm/mathreu",
		Notes:                 "Program page found; current cycle application status not confirmed on page.",
	},
	{
		OpportunityID:         "a896739a-2a3d-4b21-b6bb-10b14801682e",
		ApplicationStatus:     "unknown",
		VerificationSourceURL: "https://smconservation.gmu.edu/nsf-reu/",
		Notes:                 "Program page found; current application window not explicitly stated.",
	},
	{
		OpportunityID:         "f258cd0c-fcef-4357-b5ca-64833a386913",
		ApplicationStatus:     "unknown",
		VerificationSourceURL: "https://CoastalREU.fiu.edu",
		Notes:                 "Program site referenced; application availability not verified.",
	},
	{
		OpportunityID:         "422fef14-6abf-4928-8bef-bc65410635b9",
		ApplicationStatus:     "unknown",
		VerificationSourceURL: "https://huixue.people.clemson.edu/REU.html",
		Notes:                 "Program page found; no verified application deadline.",
	},
	{
		OpportunityID:         "ae4d661c-b087-4e2e-9ffa-4d3f94ff302e",
		ApplicationStatus:     "unknown",
		VerificationSourceURL: "https://www.amnh.org/research/richard-gilder-graduate-school/academics/fellowship-and-grant-opportunities/undergraduate-fellowships/reu-biology-program",
		Notes:                 "Official REU biology program page; current cycle status not confirmed.",
	},
	{
		OpportunityID:         "0b3f0da6-f13a-4ee1-902a-6ab621223ea4",
		ApplicationStatus:     "unknown",
		VerificationSourceURL: "http://www.jmu.edu/biology/reu",
		Notes:                 "Biology REU program page; application availability not verified.",
	},
	{
		OpportunityID:         "b2e875de-0de0-48fb-9c60-bf77d590eb4f",
		ApplicationStatus:     "unknown",
		VerificationSourceURL: "http://buee.brooklyn.cuny.edu",
		Notes:                 "BUEE program site; current application status not verified.",
	},
	{
		OpportunityID:         "cf950792-3ba4-42ec-b5ea-0af780c62ff7",
		ApplicationStatus:     "unknown",
		VerificationSourceURL: "http://www.math.emory.edu/site/cmds-reuret/",
		Notes:                 "Program page found; application window not verified.",
	},
	{
		OpportunityID:         "8eb0f543-9a7a-4ff2-a4a8-5324d3ef8cc2",
		ApplicationStatus:     "unknown",
		VerificationSourceURL: "https://sites.google.com/view/fsu-math-reu",
		Notes:                 "FSU math REU page; application availability not verified.",
	},
	{
		OpportunityID:         "14262129-0b30-4398-ba63-5331057f7f2c",
		ApplicationStatus:     "unknown",
		VerificationSourceURL: "https://sites.google.com/georgiasouthern.edu/icps-reu/",
		Notes:                 "RESCoPE program page; application status not verified.",
	},
}

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://careeros:careeros@localhost:5433/careeros?sslmode=disable"
	}
	reviewerIDStr := os.Getenv("PILOT_REVIEWER_USER_ID")
	if reviewerIDStr == "" {
		log.Fatal("PILOT_REVIEWER_USER_ID is required (UUID of admin user)")
	}
	reviewerID, err := uuid.Parse(reviewerIDStr)
	if err != nil {
		log.Fatalf("invalid PILOT_REVIEWER_USER_ID: %v", err)
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	svc := researchverification.NewService(researchverification.NewRepository(pool))

	for _, entry := range pilotEntries {
		req := researchverification.VerifyRequest{
			ApplicationStatus: entry.ApplicationStatus,
		}
		if entry.VerificationSourceURL == "" {
			req.VerificationMethod = "unknown"
		} else {
			req.VerificationMethod = "manual_official_page"
		}
		req.VerificationSourceURL = entry.VerificationSourceURL
		if entry.ApplicationURL != "" {
			req.ApplicationURL = &entry.ApplicationURL
		}
		if entry.Deadline != "" {
			req.Deadline = &entry.Deadline
		}
		if entry.OpensAt != "" {
			req.OpensAt = &entry.OpensAt
		}
		if entry.CycleLabel != "" {
			req.CycleLabel = &entry.CycleLabel
		}
		if entry.Notes != "" {
			req.Notes = &entry.Notes
		}

		oppID, err := uuid.Parse(entry.OpportunityID)
		if err != nil {
			log.Fatalf("invalid opportunity id %s: %v", entry.OpportunityID, err)
		}
		rec, err := svc.Verify(ctx, oppID, reviewerID, req)
		if err != nil {
			log.Fatalf("verify %s: %v", entry.OpportunityID, err)
		}
		fmt.Printf("verified %s -> %s (%s)\n", entry.OpportunityID, rec.ApplicationStatus, rec.ID)
	}
}
