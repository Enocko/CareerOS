-- CareerOS initial schema (Milestone 1)
-- Requires PostgreSQL 16+

CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ---------------------------------------------------------------------------
-- users
-- ---------------------------------------------------------------------------
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT users_email_unique UNIQUE (email)
);

-- ---------------------------------------------------------------------------
-- student_profiles
-- ---------------------------------------------------------------------------
CREATE TABLE student_profiles (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL,
    first_name          VARCHAR(100),
    last_name           VARCHAR(100),
    university          VARCHAR(255) DEFAULT 'Grambling State University',
    major               VARCHAR(255),
    graduation_year     INTEGER,
    career_interests    TEXT[] NOT NULL DEFAULT '{}',
    desired_roles       TEXT[] NOT NULL DEFAULT '{}',
    skills              TEXT[] NOT NULL DEFAULT '{}',
    technologies        TEXT[] NOT NULL DEFAULT '{}',
    preferred_locations TEXT[] NOT NULL DEFAULT '{}',
    work_arrangement    VARCHAR(50),
    experience_level    VARCHAR(50),
    github_url          VARCHAR(500),
    linkedin_url        VARCHAR(500),
    portfolio_url       VARCHAR(500),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT student_profiles_user_id_unique UNIQUE (user_id),
    CONSTRAINT student_profiles_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT student_profiles_graduation_year_check
        CHECK (graduation_year IS NULL OR (graduation_year >= 2020 AND graduation_year <= 2040)),
    CONSTRAINT student_profiles_work_arrangement_check
        CHECK (work_arrangement IS NULL OR work_arrangement IN ('remote', 'hybrid', 'on_site', 'flexible')),
    CONSTRAINT student_profiles_experience_level_check
        CHECK (experience_level IS NULL OR experience_level IN ('intern', 'entry', 'mid', 'senior'))
);

-- ---------------------------------------------------------------------------
-- organizations
-- ---------------------------------------------------------------------------
CREATE TABLE organizations (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         VARCHAR(255) NOT NULL,
    website_url  VARCHAR(500),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------------
-- opportunities
-- ---------------------------------------------------------------------------
CREATE TABLE opportunities (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id   UUID,
    title             VARCHAR(500) NOT NULL,
    organization_name VARCHAR(255) NOT NULL,
    description       TEXT NOT NULL,
    category          VARCHAR(50) NOT NULL,
    location          VARCHAR(255),
    work_arrangement  VARCHAR(50) NOT NULL DEFAULT 'on_site',
    deadline          DATE,
    start_date        DATE,
    eligibility       TEXT,
    skills            TEXT[] NOT NULL DEFAULT '{}',
    compensation      VARCHAR(255),
    application_url   VARCHAR(1000),
    source            VARCHAR(255) NOT NULL DEFAULT 'manual',
    status            VARCHAR(20) NOT NULL DEFAULT 'open',
    tags              TEXT[] NOT NULL DEFAULT '{}',
    last_verified     TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT opportunities_organization_id_fkey
        FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE SET NULL,
    CONSTRAINT opportunities_category_check
        CHECK (category IN (
            'internship', 'full_time', 'part_time', 'fellowship', 'scholarship',
            'research', 'hackathon', 'conference', 'apprenticeship',
            'leadership_program', 'other'
        )),
    CONSTRAINT opportunities_work_arrangement_check
        CHECK (work_arrangement IN ('remote', 'hybrid', 'on_site')),
    CONSTRAINT opportunities_status_check
        CHECK (status IN ('open', 'closed'))
);

CREATE INDEX opportunities_status_idx ON opportunities (status);
CREATE INDEX opportunities_category_idx ON opportunities (category);
CREATE INDEX opportunities_deadline_idx ON opportunities (deadline);
CREATE INDEX opportunities_work_arrangement_idx ON opportunities (work_arrangement);
CREATE INDEX opportunities_title_trgm_idx ON opportunities USING GIN (title gin_trgm_ops);

-- ---------------------------------------------------------------------------
-- saved_opportunities
-- ---------------------------------------------------------------------------
CREATE TABLE saved_opportunities (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id     UUID NOT NULL,
    opportunity_id UUID NOT NULL,
    saved_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT saved_opportunities_student_id_fkey
        FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT saved_opportunities_opportunity_id_fkey
        FOREIGN KEY (opportunity_id) REFERENCES opportunities(id) ON DELETE CASCADE,
    CONSTRAINT saved_opportunities_student_opportunity_uniq
        UNIQUE (student_id, opportunity_id)
);

CREATE INDEX saved_opportunities_student_id_idx ON saved_opportunities (student_id);

-- ---------------------------------------------------------------------------
-- applications
-- ---------------------------------------------------------------------------
CREATE TABLE applications (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id       UUID NOT NULL,
    opportunity_id   UUID NOT NULL,
    current_status   VARCHAR(50) NOT NULL DEFAULT 'saved',
    date_applied     DATE,
    notes            TEXT,
    next_action      VARCHAR(500),
    next_action_date DATE,
    interview_date   DATE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT applications_student_id_fkey
        FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT applications_opportunity_id_fkey
        FOREIGN KEY (opportunity_id) REFERENCES opportunities(id) ON DELETE CASCADE,
    CONSTRAINT applications_student_opportunity_uniq
        UNIQUE (student_id, opportunity_id),
    CONSTRAINT applications_current_status_check
        CHECK (current_status IN (
            'saved', 'preparing', 'applied', 'oa_assessment', 'interview',
            'final_round', 'offer', 'rejected', 'withdrawn', 'closed'
        ))
);

CREATE INDEX applications_student_id_idx ON applications (student_id);
CREATE INDEX applications_current_status_idx ON applications (current_status);
CREATE INDEX applications_student_status_idx ON applications (student_id, current_status);

-- ---------------------------------------------------------------------------
-- application_status_history
-- ---------------------------------------------------------------------------
CREATE TABLE application_status_history (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID NOT NULL,
    from_status    VARCHAR(50),
    to_status      VARCHAR(50) NOT NULL,
    changed_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    changed_by     UUID NOT NULL,
    CONSTRAINT application_status_history_application_id_fkey
        FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE,
    CONSTRAINT application_status_history_changed_by_fkey
        FOREIGN KEY (changed_by) REFERENCES users(id),
    CONSTRAINT application_status_history_to_status_check
        CHECK (to_status IN (
            'saved', 'preparing', 'applied', 'oa_assessment', 'interview',
            'final_round', 'offer', 'rejected', 'withdrawn', 'closed'
        ))
);

CREATE INDEX application_status_history_application_id_idx
    ON application_status_history (application_id);
