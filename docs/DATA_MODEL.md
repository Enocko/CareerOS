# CareerOS — Data Model

**Status:** Milestone 1  
**Last updated:** 2026-08-29

---

## Entity Relationship Diagram

```mermaid
erDiagram
    users ||--o| student_profiles : has
    users ||--o{ applications : owns
    users ||--o{ saved_opportunities : saves
    organizations ||--o{ opportunities : publishes
    opportunities ||--o{ saved_opportunities : "saved as"
    opportunities ||--o{ applications : "applied to"
    applications ||--o{ application_status_history : tracks

    users {
        uuid id PK
        varchar email UK
        varchar password_hash
        timestamptz created_at
        timestamptz updated_at
    }

    student_profiles {
        uuid id PK
        uuid user_id FK UK
        varchar first_name
        varchar last_name
        varchar university
        varchar major
        int graduation_year
        text_array career_interests
        text_array desired_roles
        text_array skills
        text_array technologies
        text_array preferred_locations
        varchar work_arrangement
        varchar experience_level
        varchar github_url
        varchar linkedin_url
        varchar portfolio_url
        timestamptz created_at
        timestamptz updated_at
    }

    organizations {
        uuid id PK
        varchar name
        varchar website_url
        timestamptz created_at
        timestamptz updated_at
    }

    opportunities {
        uuid id PK
        uuid organization_id FK
        varchar title
        varchar organization_name
        text description
        varchar category
        varchar location
        varchar work_arrangement
        date deadline
        date start_date
        text eligibility
        text_array skills
        varchar compensation
        varchar application_url
        varchar source
        varchar status
        text_array tags
        timestamptz last_verified
        timestamptz created_at
        timestamptz updated_at
    }

    saved_opportunities {
        uuid id PK
        uuid student_id FK
        uuid opportunity_id FK
        timestamptz saved_at
    }

    applications {
        uuid id PK
        uuid student_id FK
        uuid opportunity_id FK
        varchar current_status
        date date_applied
        text notes
        varchar next_action
        date next_action_date
        date interview_date
        timestamptz created_at
        timestamptz updated_at
    }

    application_status_history {
        uuid id PK
        uuid application_id FK
        varchar from_status
        varchar to_status
        timestamptz changed_at
        uuid changed_by FK
    }
```

---

## Tables

### `users`

Stores authentication credentials. One row per student account.

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `UUID` | PK, DEFAULT gen_random_uuid() | |
| `email` | `VARCHAR(255)` | NOT NULL, UNIQUE | Lowercased on insert |
| `password_hash` | `VARCHAR(255)` | NOT NULL | bcrypt hash, never exposed |
| `created_at` | `TIMESTAMPTZ` | NOT NULL, DEFAULT now() | |
| `updated_at` | `TIMESTAMPTZ` | NOT NULL, DEFAULT now() | |

**Indexes:**
- `users_email_idx` UNIQUE on `email` (implicit from UNIQUE constraint)

---

### `student_profiles`

One profile per user. Created on first profile update.

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `UUID` | PK | |
| `user_id` | `UUID` | NOT NULL, UNIQUE, FK → users(id) ON DELETE CASCADE | |
| `first_name` | `VARCHAR(100)` | | |
| `last_name` | `VARCHAR(100)` | | |
| `university` | `VARCHAR(255)` | DEFAULT 'Grambling State University' | |
| `major` | `VARCHAR(255)` | | |
| `graduation_year` | `INTEGER` | CHECK (graduation_year >= 2020 AND graduation_year <= 2040) | |
| `career_interests` | `TEXT[]` | DEFAULT '{}' | |
| `desired_roles` | `TEXT[]` | DEFAULT '{}' | |
| `skills` | `TEXT[]` | DEFAULT '{}' | |
| `technologies` | `TEXT[]` | DEFAULT '{}' | |
| `preferred_locations` | `TEXT[]` | DEFAULT '{}' | |
| `work_arrangement` | `VARCHAR(50)` | CHECK IN ('remote', 'hybrid', 'on_site', 'flexible') | |
| `experience_level` | `VARCHAR(50)` | CHECK IN ('intern', 'entry', 'mid', 'senior') | |
| `github_url` | `VARCHAR(500)` | | |
| `linkedin_url` | `VARCHAR(500)` | | |
| `portfolio_url` | `VARCHAR(500)` | | |
| `created_at` | `TIMESTAMPTZ` | NOT NULL, DEFAULT now() | |
| `updated_at` | `TIMESTAMPTZ` | NOT NULL, DEFAULT now() | |

**Indexes:**
- `student_profiles_user_id_idx` UNIQUE on `user_id` (implicit)

---

### `organizations`

Optional normalized organization records. Opportunities also store `organization_name` for denormalized display.

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `UUID` | PK | |
| `name` | `VARCHAR(255)` | NOT NULL | |
| `website_url` | `VARCHAR(500)` | | |
| `created_at` | `TIMESTAMPTZ` | NOT NULL, DEFAULT now() | |
| `updated_at` | `TIMESTAMPTZ` | NOT NULL, DEFAULT now() | |

---

### `opportunities`

Core opportunity catalog. Seeded with development data for Milestone 1.

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `UUID` | PK | |
| `organization_id` | `UUID` | FK → organizations(id) ON DELETE SET NULL | Optional |
| `title` | `VARCHAR(500)` | NOT NULL | |
| `organization_name` | `VARCHAR(255)` | NOT NULL | Denormalized for display |
| `description` | `TEXT` | NOT NULL | |
| `category` | `VARCHAR(50)` | NOT NULL | See enum below |
| `location` | `VARCHAR(255)` | | |
| `work_arrangement` | `VARCHAR(50)` | NOT NULL, DEFAULT 'on_site' | remote, hybrid, on_site |
| `deadline` | `DATE` | | |
| `start_date` | `DATE` | | |
| `eligibility` | `TEXT` | | |
| `skills` | `TEXT[]` | DEFAULT '{}' | |
| `compensation` | `VARCHAR(255)` | | Public info only |
| `application_url` | `VARCHAR(1000)` | | External apply link |
| `source` | `VARCHAR(255)` | DEFAULT 'manual' | How it was added |
| `status` | `VARCHAR(20)` | NOT NULL, DEFAULT 'open' | open, closed |
| `tags` | `TEXT[]` | DEFAULT '{}' | |
| `last_verified` | `TIMESTAMPTZ` | | |
| `created_at` | `TIMESTAMPTZ` | NOT NULL, DEFAULT now() | |
| `updated_at` | `TIMESTAMPTZ` | NOT NULL, DEFAULT now() | |

**Category enum:** `internship`, `full_time`, `part_time`, `fellowship`, `scholarship`, `research`, `hackathon`, `conference`, `apprenticeship`, `leadership_program`, `other`

**Indexes:**
- `opportunities_status_idx` on `status` — filter open opportunities
- `opportunities_category_idx` on `category` — filter by type
- `opportunities_deadline_idx` on `deadline` — sort/filter by deadline
- `opportunities_work_arrangement_idx` on `work_arrangement` — remote filter
- `opportunities_title_trgm_idx` GIN on `title gin_trgm_ops` — text search (requires pg_trgm extension)

---

### `saved_opportunities`

Bookmarks. One save per student per opportunity.

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `UUID` | PK | |
| `student_id` | `UUID` | NOT NULL, FK → users(id) ON DELETE CASCADE | |
| `opportunity_id` | `UUID` | NOT NULL, FK → opportunities(id) ON DELETE CASCADE | |
| `saved_at` | `TIMESTAMPTZ` | NOT NULL, DEFAULT now() | |

**Constraints:**
- `saved_opportunities_student_opportunity_uniq` UNIQUE (`student_id`, `opportunity_id`)

**Indexes:**
- `saved_opportunities_student_id_idx` on `student_id` — list saves by student

---

### `applications`

Application tracker entries. One per student per opportunity.

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `UUID` | PK | |
| `student_id` | `UUID` | NOT NULL, FK → users(id) ON DELETE CASCADE | |
| `opportunity_id` | `UUID` | NOT NULL, FK → opportunities(id) ON DELETE CASCADE | |
| `current_status` | `VARCHAR(50)` | NOT NULL, DEFAULT 'saved' | See status enum |
| `date_applied` | `DATE` | | Set when status becomes `applied` |
| `notes` | `TEXT` | | Free-form student notes |
| `next_action` | `VARCHAR(500)` | | e.g., "Follow up with recruiter" |
| `next_action_date` | `DATE` | | |
| `interview_date` | `DATE` | | |
| `created_at` | `TIMESTAMPTZ` | NOT NULL, DEFAULT now() | |
| `updated_at` | `TIMESTAMPTZ` | NOT NULL, DEFAULT now() | |

**Status enum:** `saved`, `preparing`, `applied`, `oa_assessment`, `interview`, `final_round`, `offer`, `rejected`, `withdrawn`, `closed`

**Constraints:**
- `applications_student_opportunity_uniq` UNIQUE (`student_id`, `opportunity_id`)

**Indexes:**
- `applications_student_id_idx` on `student_id` — dashboard query
- `applications_current_status_idx` on `current_status` — filter by status
- `applications_student_status_idx` on (`student_id`, `current_status`) — composite for dashboard filters

---

### `application_status_history`

Immutable audit trail of every status change.

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `UUID` | PK | |
| `application_id` | `UUID` | NOT NULL, FK → applications(id) ON DELETE CASCADE | |
| `from_status` | `VARCHAR(50)` | | NULL on initial creation |
| `to_status` | `VARCHAR(50)` | NOT NULL | |
| `changed_at` | `TIMESTAMPTZ` | NOT NULL, DEFAULT now() | |
| `changed_by` | `UUID` | NOT NULL, FK → users(id) | Who made the change |

**Indexes:**
- `application_status_history_application_id_idx` on `application_id` — history lookup

---

## Design Decisions

| Decision | Rationale |
|---|---|
| UUID primary keys | No sequential ID leakage; safe for distributed future |
| `organization_name` denormalized on opportunities | Opportunities displayable without JOIN; org table optional for Milestone 1 |
| `TEXT[]` for skills/tags | PostgreSQL native arrays; sufficient for MVP filtering |
| Unique constraint on (student_id, opportunity_id) for applications | Prevents duplicate applications at DB level |
| Status history as separate table | Immutable audit trail; supports analytics later |
| `student_id` references `users.id` directly | Simpler than separate students table for Milestone 1 |
| pg_trgm extension for search | Better than ILIKE for title search; no external search engine needed |

---

## Status Transition Rules

All statuses are valid targets. The application does not enforce a strict state machine in Milestone 1 (students may skip stages or move backward). Status history records every change regardless of direction.

When status changes to `applied` and `date_applied` is null, the service sets `date_applied` to the current date automatically.
