import type {
  ApiError,
  Application,
  AuthResponse,
  Opportunity,
  OpportunitySummary,
  PaginatedResponse,
  Profile,
  User,
} from '../types'

const API_BASE = import.meta.env.VITE_API_URL || ''

class ApiClient {
  // In-memory session token (not persisted). HttpOnly cookie is primary in production;
  // Bearer fallback keeps local dev working when frontend and API are on different ports.
  private token: string | null = null

  setToken(token: string | null) {
    this.token = token
  }

  private async request<T>(
    path: string,
    options: RequestInit = {},
  ): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string>),
    }

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`
    }

    const response = await fetch(`${API_BASE}${path}`, {
      ...options,
      headers,
      credentials: 'include',
    })

    if (response.status === 204 || response.status === 205) {
      if (!response.ok) {
        throw new Error('Request failed')
      }
      return undefined as T
    }

    const text = await response.text()
    if (!text) {
      if (!response.ok) {
        throw new Error('Request failed')
      }
      return undefined as T
    }

    const contentType = response.headers.get('content-type') ?? ''
    if (!contentType.includes('application/json')) {
      const summary = text.trim().slice(0, 120) || response.statusText || 'empty response'
      throw new Error(
        response.ok
          ? 'Server returned a non-JSON response'
          : `Request failed (${response.status}): ${summary}`,
      )
    }

    let data: T | ApiError
    try {
      data = JSON.parse(text) as T | ApiError
    } catch {
      throw new Error(`Request failed (${response.status}): invalid JSON response`)
    }

    if (!response.ok) {
      const err = data as ApiError
      throw new Error(err.error?.message || 'Request failed')
    }

    return data as T
  }

  register(email: string, password: string) {
    return this.request<AuthResponse>('/api/v1/auth/register', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    })
  }

  login(email: string, password: string) {
    return this.request<AuthResponse>('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    })
  }

  logout() {
    return this.request<void>('/api/v1/auth/logout', { method: 'POST' })
  }

  getMe() {
    return this.request<User>('/api/v1/auth/me')
  }

  getProfile() {
    return this.request<Profile>('/api/v1/profile')
  }

  updateProfile(profile: Partial<Profile>) {
    return this.request<Profile>('/api/v1/profile', {
      method: 'PUT',
      body: JSON.stringify(profile),
    })
  }

  listOpportunities(params?: Record<string, string>) {
    const query = params ? `?${new URLSearchParams(params)}` : ''
    return this.request<PaginatedResponse<OpportunitySummary>>(
      `/api/v1/opportunities${query}`,
    )
  }

  listRecommendedOpportunities(params?: Record<string, string>) {
    const query = params ? `?${new URLSearchParams(params)}` : ''
    return this.request<import('../types').RecommendedResponse>(
      `/api/v1/opportunities/recommended${query}`,
    )
  }

  recordRecommendationEvent(event: string, opportunityId?: string) {
    return this.request<void>('/api/v1/opportunities/recommended/events', {
      method: 'POST',
      body: JSON.stringify({
        event,
        opportunity_id: opportunityId,
      }),
    }).catch(() => undefined)
  }

  listNotifications(params?: Record<string, string>) {
    const query = params ? `?${new URLSearchParams(params)}` : ''
    return this.request<import('../types').NotificationsResponse>(
      `/api/v1/notifications${query}`,
    )
  }

  getUnreadNotificationCount() {
    return this.request<{ unread: number }>('/api/v1/notifications/unread-count')
  }

  markNotificationRead(id: string) {
    return this.request<void>(`/api/v1/notifications/${id}/read`, { method: 'PATCH' })
  }

  markAllNotificationsRead() {
    return this.request<{ marked_read: number }>('/api/v1/notifications/mark-all-read', {
      method: 'POST',
    })
  }

  getOpportunity(id: string) {
    return this.request<Opportunity>(`/api/v1/opportunities/${id}`)
  }

  saveOpportunity(id: string) {
    return this.request<{ id: string; opportunity_id: string; saved_at: string }>(
      `/api/v1/opportunities/${id}/save`,
      { method: 'POST' },
    )
  }

  unsaveOpportunity(id: string) {
    return this.request<void>(`/api/v1/opportunities/${id}/save`, {
      method: 'DELETE',
    })
  }

  deleteApplication(id: string) {
    return this.request<void>(`/api/v1/applications/${id}`, {
      method: 'DELETE',
    })
  }

  listSavedOpportunities(params?: Record<string, string>) {
    const query = params ? `?${new URLSearchParams(params)}` : ''
    return this.request<PaginatedResponse<OpportunitySummary>>(
      `/api/v1/saved-opportunities${query}`,
    )
  }

  createApplication(opportunityId: string, notes?: string) {
    return this.request<Application>('/api/v1/applications', {
      method: 'POST',
      body: JSON.stringify({ opportunity_id: opportunityId, notes }),
    })
  }

  listApplications(params?: Record<string, string>) {
    const query = params ? `?${new URLSearchParams(params)}` : ''
    return this.request<PaginatedResponse<Application>>(
      `/api/v1/applications${query}`,
    )
  }

  getApplication(id: string) {
    return this.request<Application>(`/api/v1/applications/${id}`)
  }

  updateApplication(
    id: string,
    data: Partial<{
      current_status: string
      notes: string | null
      next_action: string | null
      next_action_date: string | null
      interview_date: string | null
    }>,
  ) {
    return this.request<Application>(`/api/v1/applications/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    })
  }

  getResearchVerificationQueue(params?: Record<string, string>) {
    const query = params ? `?${new URLSearchParams(params)}` : ''
    return this.request<{ data: import('../types').ResearchVerificationQueueItem[] }>(
      `/api/v1/admin/research/queue${query}`,
    )
  }

  submitResearchVerification(
    opportunityId: string,
    data: {
      application_status: string
      verification_source_url: string
      verification_method: string
      application_url?: string | null
      deadline?: string | null
      opens_at?: string | null
      cycle_label?: string | null
      notes?: string | null
    },
  ) {
    return this.request<import('../types').ResearchVerificationRecord>(
      `/api/v1/admin/research/opportunities/${opportunityId}/verify`,
      {
        method: 'POST',
        body: JSON.stringify(data),
      },
    )
  }

  getResearchVerificationMetrics() {
    return this.request<import('../types').ResearchVerificationMetrics>(
      '/api/v1/admin/research/metrics',
    )
  }

  reportOpportunity(id: string, reason: string, note?: string) {
    return this.request<{ id: string }>(`/api/v1/opportunities/${id}/report`, {
      method: 'POST',
      body: JSON.stringify({ reason, note: note || undefined }),
    })
  }

  getAdminOverview() {
    return this.request<import('../types').AdminOverview>('/api/v1/admin/overview')
  }

  listOpportunityReports(params?: Record<string, string>) {
    const query = params ? `?${new URLSearchParams(params)}` : ''
    return this.request<{ data: import('../types').OpportunityReportQueueItem[] }>(
      `/api/v1/admin/reports${query}`,
    )
  }

  resolveOpportunityReports(opportunityId: string, status: 'resolved' | 'dismissed') {
    return this.request<{ updated: number }>(
      `/api/v1/admin/reports/opportunities/${opportunityId}`,
      {
        method: 'PATCH',
        body: JSON.stringify({ status }),
      },
    )
  }
}

export const api = new ApiClient()
