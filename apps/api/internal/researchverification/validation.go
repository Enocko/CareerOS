package researchverification

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/careeros/api/internal/opportunitytype"
)

// ComputeNextVerification returns when this record should be re-checked.
func ComputeNextVerification(status string, opensAt, deadline *time.Time, now time.Time) time.Time {
	now = now.UTC()
	switch status {
	case opportunitytype.ApplicationStatusOpen:
		if deadline != nil {
			nearDeadline := deadline.UTC().AddDate(0, 0, -7)
			cap := now.AddDate(0, 0, 14)
			if nearDeadline.Before(cap) {
				return nearDeadline
			}
			return cap
		}
		return now.AddDate(0, 0, 14)
	case opportunitytype.ApplicationStatusUpcoming:
		if opensAt != nil {
			check := opensAt.UTC().AddDate(0, 0, -3)
			if check.After(now) {
				return check
			}
		}
		return now.AddDate(0, 0, 7)
	case opportunitytype.ApplicationStatusClosed:
		return now.AddDate(0, 0, 180)
	default:
		return now.AddDate(0, 0, 90)
	}
}

// ValidateVerifyRequest enforces consistency rules for verification submissions.
func ValidateVerifyRequest(req VerifyRequest, now time.Time) error {
	status := strings.TrimSpace(req.ApplicationStatus)
	if !opportunitytype.ValidApplicationStatus(status) {
		return fmt.Errorf("invalid application_status: %q", status)
	}

	method := strings.TrimSpace(req.VerificationMethod)
	if method == "" {
		method = opportunitytype.AvailabilityMethodManualOfficialPage
	}
	if !opportunitytype.ValidAvailabilityMethod(method) {
		return fmt.Errorf("invalid verification_method: %q", method)
	}
	if method == opportunitytype.AvailabilityMethodNSFAwardOnly && status != opportunitytype.ApplicationStatusUnknown {
		return fmt.Errorf("nsf_award_only method is only valid for unknown status")
	}

	sourceURL := strings.TrimSpace(req.VerificationSourceURL)
	if status != opportunitytype.ApplicationStatusUnknown && sourceURL == "" {
		return fmt.Errorf("verification_source_url is required for status %q", status)
	}
	if sourceURL != "" && !isHTTPURL(sourceURL) {
		return fmt.Errorf("verification_source_url must be a valid http(s) URL")
	}

	var opensAt, deadline *time.Time
	if req.OpensAt != nil && strings.TrimSpace(*req.OpensAt) != "" {
		t, err := parseDate(*req.OpensAt)
		if err != nil {
			return fmt.Errorf("invalid opens_at: %w", err)
		}
		opensAt = &t
	}
	if req.Deadline != nil && strings.TrimSpace(*req.Deadline) != "" {
		t, err := parseDate(*req.Deadline)
		if err != nil {
			return fmt.Errorf("invalid deadline: %w", err)
		}
		deadline = &t
	}

	if opensAt != nil && deadline != nil && deadline.Before(*opensAt) {
		return fmt.Errorf("deadline must be on or after opens_at")
	}

	appURL := ""
	if req.ApplicationURL != nil {
		appURL = strings.TrimSpace(*req.ApplicationURL)
	}

	switch status {
	case opportunitytype.ApplicationStatusOpen:
		if appURL == "" {
			return fmt.Errorf("application_url is required when status is open")
		}
		if !isHTTPURL(appURL) {
			return fmt.Errorf("application_url must be a valid http(s) URL")
		}
		if isGenericETAP(appURL) || isNSFAwardURL(appURL) {
			return fmt.Errorf("application_url must be a specific application destination, not a generic page")
		}
	case opportunitytype.ApplicationStatusUpcoming, opportunitytype.ApplicationStatusClosed:
		if appURL != "" && (!isHTTPURL(appURL) || isGenericETAP(appURL) || isNSFAwardURL(appURL)) {
			return fmt.Errorf("application_url must be a specific valid application destination when provided")
		}
	case opportunitytype.ApplicationStatusUnknown:
		if appURL != "" {
			return fmt.Errorf("application_url must not be set when status is unknown")
		}
	}

	if status == opportunitytype.ApplicationStatusClosed && deadline != nil {
		today := now.UTC().Truncate(24 * time.Hour)
		if deadline.After(today) {
			return fmt.Errorf("closed status cannot have a future application deadline")
		}
	}

	return nil
}

func parseDate(raw string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(raw))
}

func isHTTPURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func isGenericETAP(raw string) bool {
	lower := strings.ToLower(strings.TrimRight(strings.TrimSpace(raw), "/"))
	return lower == "https://etap.nsf.gov" || lower == "http://etap.nsf.gov"
}

func isNSFAwardURL(raw string) bool {
	lower := strings.ToLower(raw)
	return strings.Contains(lower, "nsf.gov/awardsearch") || strings.Contains(lower, "nsf.gov/award")
}
