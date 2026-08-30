-- Tracks student detail views so closed/stale listings remain reachable via prior access.
CREATE TABLE opportunity_views (
    student_id     UUID NOT NULL,
    opportunity_id UUID NOT NULL,
    last_viewed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT opportunity_views_student_id_fkey
        FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT opportunity_views_opportunity_id_fkey
        FOREIGN KEY (opportunity_id) REFERENCES opportunities(id) ON DELETE CASCADE,
    PRIMARY KEY (student_id, opportunity_id)
);

CREATE INDEX opportunity_views_opportunity_id_idx ON opportunity_views (opportunity_id);
