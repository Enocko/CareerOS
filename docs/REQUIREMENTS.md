# CareerOS — Requirements

**Status:** Milestone 1  
**Last updated:** 2026-08-29

---

## Functional Requirements

### FR-1: Authentication

| ID | Requirement |
|---|---|
| FR-1.1 | A student can register with email and password |
| FR-1.2 | A student can log in with email and password |
| FR-1.3 | A student can log out (token invalidated client-side; server validates expiry) |
| FR-1.4 | Passwords are hashed with bcrypt before storage |
| FR-1.5 | Duplicate email registration is rejected |
| FR-1.6 | Authenticated endpoints reject unauthenticated requests with 401 |
| FR-1.7 | Password must be at least 8 characters |

### FR-2: Student Profile

| ID | Requirement |
|---|---|
| FR-2.1 | An authenticated student can view their profile |
| FR-2.2 | An authenticated student can create/update their profile |
| FR-2.3 | Profile fields: name, university, major, graduation year, career interests, desired roles, skills, technologies, preferred locations, work arrangement, experience level, optional GitHub/LinkedIn/portfolio URLs |
| FR-2.4 | Profile is created automatically on first update if it does not exist |
| FR-2.5 | Only the profile owner can read or modify their profile |

### FR-3: Opportunity Discovery

| ID | Requirement |
|---|---|
| FR-3.1 | An authenticated student can browse a paginated list of opportunities |
| FR-3.2 | An authenticated student can search opportunities by title, organization, or description |
| FR-3.3 | An authenticated student can filter by category, location, work arrangement, and remote |
| FR-3.4 | An authenticated student can view full opportunity details |
| FR-3.5 | Only open opportunities are shown by default |
| FR-3.6 | Opportunities include: title, organization, description, category, location, work arrangement, deadline, skills, application URL, tags |

### FR-4: Save Opportunity

| ID | Requirement |
|---|---|
| FR-4.1 | An authenticated student can save an opportunity |
| FR-4.2 | An authenticated student can unsave an opportunity |
| FR-4.3 | An authenticated student can list their saved opportunities |
| FR-4.4 | Saving the same opportunity twice is idempotent (no duplicate) |
| FR-4.5 | Only the saving student can see their saved list |

### FR-5: Application Tracker

| ID | Requirement |
|---|---|
| FR-5.1 | An authenticated student can create an application for an opportunity |
| FR-5.2 | An authenticated student can update application status |
| FR-5.3 | Every status change is recorded in application status history with timestamp |
| FR-5.4 | An authenticated student can add notes to an application |
| FR-5.5 | An authenticated student can view their application dashboard (all applications) |
| FR-5.6 | An authenticated student can view a single application with status history |
| FR-5.7 | Only one application per student per opportunity |
| FR-5.8 | Valid statuses: `saved`, `preparing`, `applied`, `oa_assessment`, `interview`, `final_round`, `offer`, `rejected`, `withdrawn`, `closed` |
| FR-5.9 | Only the application owner can read or modify their applications |

### FR-6: Health Check

| ID | Requirement |
|---|---|
| FR-6.1 | API exposes `GET /health` returning service status and database connectivity |

---

## Non-Functional Requirements

### NFR-1: Security

| ID | Requirement |
|---|---|
| NFR-1.1 | No secrets committed to source control |
| NFR-1.2 | Configuration via environment variables |
| NFR-1.3 | Passwords never logged or returned in API responses |
| NFR-1.4 | JWT tokens signed with a configurable secret |
| NFR-1.5 | Input validation on all API endpoints |
| NFR-1.6 | SQL injection prevented via parameterized queries |
| NFR-1.7 | CORS configured for local frontend origin |

### NFR-2: Reliability

| ID | Requirement |
|---|---|
| NFR-2.1 | Database constraints enforce data integrity (FK, unique, not null) |
| NFR-2.2 | Transactions used for multi-step writes (e.g., status change + history) |
| NFR-2.3 | Consistent error response format across all endpoints |

### NFR-3: Observability

| ID | Requirement |
|---|---|
| NFR-3.1 | Structured JSON logging with request ID, method, path, status, duration |
| NFR-3.2 | Health endpoint reports database connectivity |

### NFR-4: Maintainability

| ID | Requirement |
|---|---|
| NFR-4.1 | Code organized by domain within a modular monolith |
| NFR-4.2 | Database schema managed via versioned SQL migrations |
| NFR-4.3 | API versioned under `/api/v1` |

### NFR-5: Testability

| ID | Requirement |
|---|---|
| NFR-5.1 | Unit tests for domain logic (validation, status transitions) |
| NFR-5.2 | Integration tests for API + database flows |
| NFR-5.3 | Tests runnable without external services beyond PostgreSQL |

### NFR-6: Local Development

| ID | Requirement |
|---|---|
| NFR-6.1 | Full stack runnable via Docker Compose |
| NFR-6.2 | Seed data clearly labeled as development data |
| NFR-6.3 | `.env.example` documents all required environment variables |

---

## MVP Acceptance Criteria

The milestone is complete when all of the following pass:

### End-to-End Workflow

- [ ] Student registers with email and password
- [ ] Student logs in and receives a JWT
- [ ] Student creates/updates their profile
- [ ] Student browses opportunities (seed data visible)
- [ ] Student views opportunity details
- [ ] Student saves an opportunity
- [ ] Student creates an application for that opportunity
- [ ] Student updates application status to `applied`
- [ ] Student views application dashboard showing the application with correct status
- [ ] Status history shows the transition with timestamp

### Technical

- [ ] `GET /health` returns 200 with database connected
- [ ] Unauthenticated access to protected endpoints returns 401
- [ ] Duplicate email registration returns 409
- [ ] Duplicate application for same opportunity returns 409
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] `docker compose up` starts API, frontend, and PostgreSQL
- [ ] No secrets in repository

### Documentation

- [ ] `docs/PRODUCT.md` complete
- [ ] `docs/REQUIREMENTS.md` complete
- [ ] `docs/ARCHITECTURE.md` complete
- [ ] `docs/DATA_MODEL.md` complete
- [ ] `docs/API.md` complete
