-- Background jobs and in-app notifications (CareerOS v1).

CREATE TABLE background_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_type        VARCHAR(80) NOT NULL,
    payload         JSONB NOT NULL DEFAULT '{}',
    status          VARCHAR(20) NOT NULL DEFAULT 'queued',
    idempotency_key VARCHAR(255) NOT NULL,
    attempts        INTEGER NOT NULL DEFAULT 0,
    max_attempts    INTEGER NOT NULL DEFAULT 5,
    run_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    locked_at       TIMESTAMPTZ,
    locked_by       VARCHAR(100),
    last_error      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at    TIMESTAMPTZ,
    CONSTRAINT background_jobs_status_check
        CHECK (status IN ('queued', 'processing', 'completed', 'retryable', 'failed'))
);

CREATE UNIQUE INDEX background_jobs_active_idempotency_idx
    ON background_jobs (idempotency_key)
    WHERE status IN ('queued', 'processing', 'retryable');

CREATE INDEX background_jobs_claim_idx
    ON background_jobs (run_at)
    WHERE status IN ('queued', 'retryable');

CREATE INDEX background_jobs_status_idx ON background_jobs (status);

CREATE TABLE notifications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type            VARCHAR(80) NOT NULL,
    title           VARCHAR(255) NOT NULL,
    message         TEXT NOT NULL,
    opportunity_id  UUID REFERENCES opportunities(id) ON DELETE SET NULL,
    application_id  UUID REFERENCES applications(id) ON DELETE SET NULL,
    idempotency_key VARCHAR(255) NOT NULL UNIQUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    read_at         TIMESTAMPTZ
);

CREATE INDEX notifications_user_created_idx ON notifications (user_id, created_at DESC);
CREATE INDEX notifications_user_unread_idx ON notifications (user_id) WHERE read_at IS NULL;

COMMENT ON TABLE background_jobs IS 'Durable at-least-once background work queue';
COMMENT ON TABLE notifications IS 'In-app student notifications';
