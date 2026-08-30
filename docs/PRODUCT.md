# CareerOS — Product Overview

**Status:** Milestone 1  
**Last updated:** 2026-08-29

---

## Problem Statement

HBCU students discover career opportunities across many disconnected sources: company career pages, university portals, student organizations, conferences, fellowships, hackathons, and public feeds. Deadlines are easy to miss, and application tracking is often handled manually through spreadsheets, notes, browser tabs, or memory.

There is no centralized, student-first platform designed around the specific workflows and needs of HBCU students — especially those at institutions like Grambling State University.

**CareerOS** solves this by centralizing opportunity discovery and application tracking into a single career operating system.

---

## Target Users

### Primary (Milestone 1)

- **HBCU undergraduate students** seeking internships, jobs, fellowships, scholarships, and other career opportunities
- Initial focus: **Grambling State University** students
- Example persona: Computer Science major, looking for SWE internships, needs to track many applications across fragmented platforms

### Future (Out of Scope for Milestone 1)

- Student organizations
- University career centers
- Employers and nonprofits
- Program and hackathon organizers

---

## Core User Journey

```text
Create account
    ↓
Build career profile
    ↓
Discover opportunities (browse, search, filter)
    ↓
View opportunity details
    ↓
Save opportunity
    ↓
Apply externally (via application URL)
    ↓
Create application tracker entry
    ↓
Update application status through pipeline
    ↓
View application dashboard
```

### Application Pipeline

```text
Saved → Preparing → Applied → OA/Assessment → Interview → Final Round → Offer
```

Terminal states: Rejected, Withdrawn, Closed

---

## MVP Scope (Milestone 1)

The first vertical slice delivers:

| Capability | Description |
|---|---|
| Authentication | Register, login, logout, secure password handling |
| Student profile | Create and update career profile |
| Opportunity discovery | Browse, search, filter, view details |
| Save opportunity | Bookmark opportunities for later |
| Application tracker | Create application, update status, view history |
| Application dashboard | View all tracked applications with current status |
| Health check | API liveness/readiness endpoint |

---

## Explicitly Out of Scope (Milestone 1)

The following are **not** part of this milestone:

- Redis, Kafka/NATS, or any message broker
- Microservices or service extraction
- Kubernetes, AWS, or cloud deployment
- OpenSearch/Elasticsearch
- AI recommendations or ML matching
- Opportunity ingestion pipelines or web scraping
- Notifications and deadline reminders (async or sync)
- Application analytics and conversion metrics
- Organization submissions and moderation
- Admin interface
- OAuth / Google / Microsoft sign-in
- University email verification
- Resume upload or management
- Mobile app or browser extension
- Rate limiting via Redis (in-memory or DB-based limits acceptable later)

---

## Success Criteria (Milestone 1)

A student at Grambling State University can:

1. Register and log in
2. Complete their career profile
3. Browse seeded development opportunities
4. Save an opportunity
5. Create an application and set its status to "Applied"
6. View their application dashboard with status history

All of the above works end-to-end through the React frontend and Go API.

---

## Differentiation

CareerOS is **not** LinkedIn. It is career infrastructure designed specifically around HBCU student workflows: centralized discovery, application tracking, deadline awareness, and eventually HBCU-specific opportunity sources.

---

## Validation Plan

1. **Phase 1 (ongoing):** Interview Grambling students about current workflows
2. **Phase 2 (this milestone):** Build and test the vertical slice locally
3. **Phase 3 (next):** Recruit 5–10 pilot users
4. **Phase 4:** Observe real usage and iterate
