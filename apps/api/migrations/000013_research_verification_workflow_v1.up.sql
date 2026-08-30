-- Research Verification Workflow v1: audit trail for application availability verifications

CREATE TABLE research_availability_verifications (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    opportunity_id          UUID NOT NULL REFERENCES opportunities(id) ON DELETE CASCADE,
    application_status      VARCHAR(20) NOT NULL,
    application_url         TEXT,
    verification_source_url TEXT NOT NULL,
    opens_at                DATE,
    deadline                DATE,
    cycle_label             VARCHAR(100),
    verification_method     VARCHAR(50) NOT NULL,
    verified_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    verified_by             UUID REFERENCES users(id) ON DELETE SET NULL,
    next_verification_at    TIMESTAMPTZ,
    notes                   TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT research_avail_verifications_status_check
        CHECK (application_status IN ('open', 'upcoming', 'closed', 'unknown')),
    CONSTRAINT research_avail_verifications_method_check
        CHECK (verification_method IN (
            'manual_official_page', 'manual_verified', 'automated_official',
            'partner_verified', 'unknown', 'nsf_award_only'
        ))
);

CREATE INDEX research_avail_verifications_opportunity_idx
    ON research_availability_verifications (opportunity_id, verified_at DESC);

COMMENT ON TABLE research_availability_verifications IS
    'Append-only audit trail for research application availability verifications.';
