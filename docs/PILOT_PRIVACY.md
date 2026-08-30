# CareerOS Pilot Privacy Note

**Audience:** Pilot students and operators (5–10 user pilot at Grambling State University).

**Not legal advice.** This is an operational summary of what CareerOS stores during the pilot.

## What we store

| Data | Purpose |
|------|---------|
| Email + password hash | Account authentication |
| Profile preferences | Recommendations (major, interests, skills, locations, roles) |
| Saved opportunities | Bookmarked listings |
| Application tracking | Employment application status, notes, deadlines |
| Notifications | In-app deadline reminders |
| Opportunity views | Allow access to closed listings you previously viewed |
| Issue reports | Student-submitted listing quality reports |

## What we do not store

- Social Security numbers
- Demographic survey data beyond profile preferences
- Payment information
- OAuth tokens from third parties

## Third-party data

Opportunity listings are ingested from public sources (USAJobs, employer ATS boards, NSF). CareerOS displays source attribution and verification status.

## Access controls

- Students can only access their own profile, saves, applications, and notifications.
- Admin routes require email allowlist (`CAREEROS_ADMIN_EMAILS`).

## Deletion requests

Contact the pilot operator. Account deletion removes the user record and cascades related data (saves, applications, notifications, reports).

## Security measures (pilot)

- Passwords hashed with bcrypt (cost 12)
- Session tokens in HttpOnly cookies (production)
- HTTPS required in production
- Rate limiting on login/register/report endpoints
