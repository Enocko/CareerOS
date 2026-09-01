export interface User {
  id: string
  email: string
  created_at: string
}

export interface AuthResponse {
  user: User
  token: string
}

export interface Profile {
  id: string
  user_id: string
  first_name: string | null
  last_name: string | null
  university: string | null
  major: string | null
  graduation_year: number | null
  career_interests: string[]
  desired_roles: string[]
  skills: string[]
  technologies: string[]
  preferred_locations: string[]
  work_arrangement: string | null
  experience_level: string | null
  github_url: string | null
  linkedin_url: string | null
  portfolio_url: string | null
  created_at: string
  updated_at: string
}

export interface OpportunitySummary {
  id: string
  title: string
  organization_name: string
  category: string
  opportunity_type: string
  verification_method?: string | null
  employment_mode?: string | null
  location: string | null
  work_arrangement: string
  deadline: string | null
  skills: string[]
  tags: string[]
  status: string
  verification_status: string
  source_name: string
  last_checked_at: string | null
  is_saved: boolean
  has_application?: boolean
  experience_level?: string | null
  career_family?: string | null
  relevance_tier?: string | null
  type_metadata?: Record<string, unknown>
  listing_count?: number
}

export interface ResearchTypeMetadata {
  research_area?: string
  stipend?: string
  housing_provided?: boolean
  travel_support?: boolean
  citizenship_required?: string
  duration_weeks?: number
  program_start?: string
  program_end?: string
  program_url?: string
  application_status?: 'open' | 'upcoming' | 'closed' | 'unknown'
  application_status_method?: string
  availability_verification_method?: string
  application_verified_at?: string
  application_verification_source_url?: string
  cycle_label?: string
  opens_at?: string
  next_verification_at?: string
}

export interface ResearchVerificationQueueItem {
  id: string
  title: string
  organization_name: string
  source_url?: string | null
  program_url?: string | null
  application_status: string
  priority_score: number
}

export interface ResearchVerificationRecord {
  id: string
  opportunity_id: string
  application_status: string
  verification_source_url?: string | null
  verified_at: string
}

export interface ResearchVerificationMetrics {
  candidate_nsf_reu_programs: number
  availability_unknown: number
  verified_programs: number
  applications_open: number
  applications_upcoming: number
  applications_closed: number
  direct_application_urls_verified: number
  verified_deadlines: number
  verification_stale: number
}

export interface ScoringFactor {
  key: string
  label: string
  points: number
  max: number
}

export type OpportunityReportReason =
  | 'appears_closed'
  | 'broken_link'
  | 'incorrect_deadline'
  | 'duplicate'
  | 'incorrect_info'
  | 'other'

export interface OpportunityReportQueueItem {
  opportunity_id: string
  opportunity_title: string
  organization_name: string
  source_name: string
  source_url?: string | null
  opportunity_status: string
  verification_status: string
  last_checked_at?: string | null
  report_count: number
  latest_report_at: string
  reasons: string[]
}

export interface AdminOverview {
  catalog_metrics: {
    employment_visible: number
    research_candidate: number
    research_open: number
    research_upcoming: number
    research_unknown: number
    research_closed: number
    verified_listings: number
    stale_listings: number
    closed_listings: number
    pending_reports: number
  }
  pending_reports: number
  duplicate_groups: number
  excess_duplicate_records: number
  source_health: {
    source_name: string
    adapter: string
    enabled: boolean
    last_success_at?: string | null
    last_run_status?: string | null
    verified_open_count: number
    consecutive_fails: number
  }[]
  job_metrics: {
    queued: number
    processing: number
    failed: number
  }
  research_metrics: ResearchVerificationMetrics
}

export interface RecommendedOpportunity {
  opportunity: OpportunitySummary
  match_score: number
  factors: ScoringFactor[]
  match_reasons: string[]
  match_summary?: string
}

export interface RecommendationMeta {
  profile_complete: boolean
  has_profile: boolean
  catalog_scored: number
  eligible_count: number
}

export interface RecommendedResponse {
  data: RecommendedOpportunity[]
  pagination: Pagination
  meta: RecommendationMeta
}

export interface NotificationItem {
  id: string
  type: string
  title: string
  message: string
  opportunity_id: string | null
  application_id: string | null
  created_at: string
  read_at: string | null
}

export interface NotificationsResponse {
  data: NotificationItem[]
  unread: number
  pagination: Pagination
}

export interface SourceAttribution {
  name: string
  source_url: string | null
}

export interface Opportunity extends OpportunitySummary {
  description: string
  type_metadata?: Record<string, unknown>
  start_date: string | null
  eligibility: string | null
  compensation: string | null
  application_url: string | null
  source: SourceAttribution
  last_seen_at: string | null
  created_at: string
  updated_at: string
  has_application: boolean
}

export interface OpportunityBrief {
  id: string
  title: string
  organization_name: string
  category: string
  deadline: string | null
  application_url?: string | null
}

export interface Application {
  id: string
  opportunity_id: string
  current_status: string
  date_applied: string | null
  notes: string | null
  next_action: string | null
  next_action_date: string | null
  interview_date: string | null
  created_at: string
  updated_at: string
  opportunity?: OpportunityBrief
  status_history?: StatusHistoryEntry[]
}

export interface StatusHistoryEntry {
  id: string
  from_status: string | null
  to_status: string
  changed_at: string
}

export interface Pagination {
  page: number
  per_page: number
  total: number
  total_pages: number
}

export interface PaginatedResponse<T> {
  data: T[]
  pagination: Pagination
}

export interface ApiError {
  error: {
    code: string
    message: string
    details?: { field: string; message: string }[]
  }
}

export const APPLICATION_STATUSES = [
  'saved',
  'preparing',
  'applied',
  'oa_assessment',
  'interview',
  'final_round',
  'offer',
  'rejected',
  'withdrawn',
  'closed',
] as const

export type ApplicationStatus = (typeof APPLICATION_STATUSES)[number]
