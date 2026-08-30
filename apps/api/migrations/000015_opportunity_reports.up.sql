-- Student opportunity issue reports and operator triage.

CREATE TABLE opportunity_reports (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    opportunity_id  UUID NOT NULL REFERENCES opportunities(id) ON DELETE CASCADE,
    reporter_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason          TEXT NOT NULL,
    note            VARCHAR(500),
    status          TEXT NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at     TIMESTAMPTZ,
    resolved_by     UUID REFERENCES users(id),
    CONSTRAINT opportunity_reports_reason_check CHECK (
        reason IN (
            'appears_closed',
            'broken_link',
            'incorrect_deadline',
            'duplicate',
            'incorrect_info',
            'other'
        )
    ),
    CONSTRAINT opportunity_reports_status_check CHECK (
        status IN ('pending', 'resolved', 'dismissed')
    )
);

CREATE INDEX opportunity_reports_opportunity_id_idx ON opportunity_reports (opportunity_id);
CREATE INDEX opportunity_reports_status_idx ON opportunity_reports (status);
CREATE INDEX opportunity_reports_created_at_idx ON opportunity_reports (created_at DESC);
