# CareerOS — API Contract

**Status:** Milestone 1  
**Base URL:** `http://localhost:8080`  
**Version prefix:** `/api/v1`  
**Last updated:** 2026-08-29

---

## Conventions

### Authentication

Protected endpoints require:

```http
Authorization: Bearer <jwt_token>
```

### Pagination

List endpoints accept:

| Param | Type | Default | Max |
|---|---|---|---|
| `page` | integer | 1 | — |
| `per_page` | integer | 20 | 100 |

Paginated response envelope:

```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 42,
    "total_pages": 3
  }
}
```

### Error Response

All errors return:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Human-readable description",
    "details": [
      { "field": "email", "message": "must be a valid email address" }
    ]
  }
}
```

### Error Codes

| HTTP Status | Code | When |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Invalid input |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 403 | `FORBIDDEN` | Valid token but not resource owner |
| 404 | `NOT_FOUND` | Resource does not exist |
| 409 | `CONFLICT` | Duplicate email, application, etc. |
| 500 | `INTERNAL_ERROR` | Unexpected server error |

---

## Endpoints

### Health

#### `GET /health`

No authentication required.

**Response 200:**

```json
{
  "status": "ok",
  "database": "connected"
}
```

**Response 503** (database unreachable):

```json
{
  "status": "degraded",
  "database": "disconnected"
}
```

---

### Authentication

#### `POST /api/v1/auth/register`

**Request:**

```json
{
  "email": "student@gram.edu",
  "password": "securepass123"
}
```

**Validation:**
- `email`: required, valid email format, max 255 chars
- `password`: required, min 8 chars, max 128 chars

**Response 201:**

```json
{
  "user": {
    "id": "uuid",
    "email": "student@gram.edu",
    "created_at": "2026-08-29T12:00:00Z"
  },
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response 409:** Email already registered.

---

#### `POST /api/v1/auth/login`

**Request:**

```json
{
  "email": "student@gram.edu",
  "password": "securepass123"
}
```

**Response 200:**

```json
{
  "user": {
    "id": "uuid",
    "email": "student@gram.edu",
    "created_at": "2026-08-29T12:00:00Z"
  },
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response 401:** Invalid email or password (same message for both to prevent enumeration).

---

#### `GET /api/v1/auth/me`

**Auth required.**

**Response 200:**

```json
{
  "id": "uuid",
  "email": "student@gram.edu",
  "created_at": "2026-08-29T12:00:00Z"
}
```

---

### Student Profile

#### `GET /api/v1/profile`

**Auth required.**

**Response 200:**

```json
{
  "id": "uuid",
  "user_id": "uuid",
  "first_name": "Jordan",
  "last_name": "Smith",
  "university": "Grambling State University",
  "major": "Computer Science",
  "graduation_year": 2027,
  "career_interests": ["backend", "cloud"],
  "desired_roles": ["Software Engineer Intern"],
  "skills": ["Python", "Go"],
  "technologies": ["AWS", "PostgreSQL"],
  "preferred_locations": ["Remote", "Dallas, TX"],
  "work_arrangement": "remote",
  "experience_level": "intern",
  "github_url": "https://github.com/jordan",
  "linkedin_url": null,
  "portfolio_url": null,
  "created_at": "2026-08-29T12:00:00Z",
  "updated_at": "2026-08-29T12:00:00Z"
}
```

**Response 404:** Profile not yet created.

---

#### `PUT /api/v1/profile`

**Auth required.** Creates profile if it does not exist (upsert).

**Request:**

```json
{
  "first_name": "Jordan",
  "last_name": "Smith",
  "university": "Grambling State University",
  "major": "Computer Science",
  "graduation_year": 2027,
  "career_interests": ["backend", "cloud"],
  "desired_roles": ["Software Engineer Intern"],
  "skills": ["Python", "Go"],
  "technologies": ["AWS", "PostgreSQL"],
  "preferred_locations": ["Remote", "Dallas, TX"],
  "work_arrangement": "remote",
  "experience_level": "intern",
  "github_url": "https://github.com/jordan",
  "linkedin_url": null,
  "portfolio_url": null
}
```

**Validation:**
- `first_name`, `last_name`: max 100 chars
- `graduation_year`: integer 2020–2040
- `work_arrangement`: one of `remote`, `hybrid`, `on_site`, `flexible`
- `experience_level`: one of `intern`, `entry`, `mid`, `senior`
- URL fields: valid URL format if provided

**Response 200:** Returns updated profile (same shape as GET).

---

### Opportunities

#### `GET /api/v1/opportunities`

**Auth required.**

**Query parameters:**

| Param | Type | Description |
|---|---|---|
| `q` | string | Search title, organization, description |
| `category` | string | Filter by category |
| `work_arrangement` | string | Filter by remote/hybrid/on_site |
| `location` | string | Filter by location (partial match) |
| `page` | integer | Page number |
| `per_page` | integer | Items per page |

**Response 200:**

```json
{
  "data": [
    {
      "id": "uuid",
      "title": "Software Engineering Intern",
      "organization_name": "Acme Corp",
      "category": "internship",
      "location": "Remote",
      "work_arrangement": "remote",
      "deadline": "2026-10-15",
      "skills": ["Python", "Go"],
      "tags": ["backend", "paid"],
      "status": "open",
      "is_saved": false
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 15,
    "total_pages": 1
  }
}
```

Note: `is_saved` indicates whether the authenticated student has saved this opportunity.

---

#### `GET /api/v1/opportunities/:id`

**Auth required.**

**Response 200:**

```json
{
  "id": "uuid",
  "title": "Software Engineering Intern",
  "organization_name": "Acme Corp",
  "description": "Full description...",
  "category": "internship",
  "location": "Remote",
  "work_arrangement": "remote",
  "deadline": "2026-10-15",
  "start_date": "2026-06-01",
  "eligibility": "Currently enrolled undergraduate",
  "skills": ["Python", "Go"],
  "compensation": "$25/hr",
  "application_url": "https://acme.com/apply",
  "source": "manual",
  "status": "open",
  "tags": ["backend", "paid"],
  "last_verified": "2026-08-01T00:00:00Z",
  "created_at": "2026-08-01T00:00:00Z",
  "updated_at": "2026-08-01T00:00:00Z",
  "is_saved": false,
  "has_application": false
}
```

**Response 404:** Opportunity not found.

---

### Saved Opportunities

#### `POST /api/v1/opportunities/:id/save`

**Auth required.** Idempotent — saving twice returns 200, not 409.

**Response 200:**

```json
{
  "id": "uuid",
  "opportunity_id": "uuid",
  "saved_at": "2026-08-29T12:00:00Z"
}
```

**Response 404:** Opportunity not found.

---

#### `DELETE /api/v1/opportunities/:id/save`

**Auth required.**

**Response 204:** No content.

**Response 404:** Save not found.

---

#### `GET /api/v1/saved-opportunities`

**Auth required.**

**Response 200:** Paginated list (same opportunity summary shape as browse, with `is_saved: true`).

---

### Applications

#### `POST /api/v1/applications`

**Auth required.**

**Request:**

```json
{
  "opportunity_id": "uuid",
  "notes": "Found via CareerOS seed data"
}
```

**Validation:**
- `opportunity_id`: required, valid UUID, must reference existing open opportunity

**Response 201:**

```json
{
  "id": "uuid",
  "opportunity_id": "uuid",
  "current_status": "saved",
  "notes": "Found via CareerOS seed data",
  "date_applied": null,
  "next_action": null,
  "next_action_date": null,
  "interview_date": null,
  "created_at": "2026-08-29T12:00:00Z",
  "updated_at": "2026-08-29T12:00:00Z",
  "opportunity": {
    "id": "uuid",
    "title": "Software Engineering Intern",
    "organization_name": "Acme Corp",
    "category": "internship",
    "deadline": "2026-10-15"
  }
}
```

Creates initial status history entry: `null → saved`.

**Response 409:** Application already exists for this opportunity.

---

#### `GET /api/v1/applications`

**Auth required.** Application dashboard.

**Query parameters:**

| Param | Type | Description |
|---|---|---|
| `status` | string | Filter by current_status |
| `page`, `per_page` | integer | Pagination |

**Response 200:**

```json
{
  "data": [
    {
      "id": "uuid",
      "opportunity_id": "uuid",
      "current_status": "applied",
      "date_applied": "2026-08-29",
      "notes": "Found via CareerOS seed data",
      "next_action": null,
      "next_action_date": null,
      "interview_date": null,
      "created_at": "2026-08-29T12:00:00Z",
      "updated_at": "2026-08-29T12:00:00Z",
      "opportunity": {
        "id": "uuid",
        "title": "Software Engineering Intern",
        "organization_name": "Acme Corp",
        "category": "internship",
        "deadline": "2026-10-15",
        "application_url": "https://acme.com/apply"
      }
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 1,
    "total_pages": 1
  }
}
```

---

#### `GET /api/v1/applications/:id`

**Auth required.** Single application with status history.

**Response 200:**

```json
{
  "id": "uuid",
  "opportunity_id": "uuid",
  "current_status": "applied",
  "date_applied": "2026-08-29",
  "notes": "Found via CareerOS seed data",
  "next_action": null,
  "next_action_date": null,
  "interview_date": null,
  "created_at": "2026-08-29T12:00:00Z",
  "updated_at": "2026-08-29T12:00:00Z",
  "opportunity": { "...": "..." },
  "status_history": [
    {
      "id": "uuid",
      "from_status": null,
      "to_status": "saved",
      "changed_at": "2026-08-29T12:00:00Z"
    },
    {
      "id": "uuid",
      "from_status": "saved",
      "to_status": "applied",
      "changed_at": "2026-08-29T12:05:00Z"
    }
  ]
}
```

**Response 403:** Application belongs to another user.
**Response 404:** Application not found.

---

#### `PATCH /api/v1/applications/:id`

**Auth required.** Update application status and/or metadata.

**Request:**

```json
{
  "current_status": "applied",
  "notes": "Submitted application via company portal",
  "next_action": "Wait for response",
  "next_action_date": "2026-09-15",
  "interview_date": null
}
```

**Validation:**
- `current_status`: if provided, must be a valid status enum value
- At least one field must be provided

**Response 200:** Returns updated application (same shape as GET single, without status_history).

Side effects:
- Status change creates a `application_status_history` row
- If status changes to `applied` and `date_applied` is null, sets `date_applied` to today

**Response 403:** Application belongs to another user.

---

## Authorization Rules

| Resource | Rule |
|---|---|
| Profile | Owner only (matched by JWT user ID) |
| Saved opportunities | Owner only |
| Applications | Owner only |
| Opportunities | Any authenticated user can read |
| Auth endpoints | Public (register/login) |
