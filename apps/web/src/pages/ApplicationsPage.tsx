import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { Application } from '../types'
import { APPLICATION_STATUSES } from '../types'
import { formatDate, formatStatus } from '../utils/format'

export function ApplicationsPage() {
  const [applications, setApplications] = useState<Application[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [statusFilter, setStatusFilter] = useState('')

  useEffect(() => {
    loadApplications()
  }, [])

  async function loadApplications(status?: string) {
    setLoading(true)
    setError('')
    setApplications([])
    try {
      const params: Record<string, string> = {}
      if (status) params.status = status
      const resp = await api.listApplications(params)
      setApplications(resp.data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load applications')
    } finally {
      setLoading(false)
    }
  }

  function handleFilterChange(e: React.ChangeEvent<HTMLSelectElement>) {
    const value = e.target.value
    setStatusFilter(value)
    loadApplications(value || undefined)
  }

  return (
    <div>
      <div className="page-header">
        <h1>Application Dashboard</h1>
        <p className="subtitle">Track your applications from saved to offer.</p>
      </div>

      <div className="card filters">
        <label>
          Filter by status
          <select value={statusFilter} onChange={handleFilterChange}>
            <option value="">All statuses</option>
            {APPLICATION_STATUSES.map((s) => (
              <option key={s} value={s}>
                {formatStatus(s)}
              </option>
            ))}
          </select>
        </label>
      </div>

      {error && <div className="alert alert-error">{error}</div>}
      {loading && <div className="loading">Loading applications...</div>}

      {!loading && applications.length === 0 && (
        <div className="empty-state">
          No applications yet.{' '}
          <Link to="/browse">Browse opportunities</Link> to get started.
        </div>
      )}

      <div className="card-list">
        {applications.map((app) => (
          <Link
            key={app.id}
            to={`/applications/${app.id}`}
            className="card opp-card"
          >
            <div className="opp-card-header">
              <h2>{app.opportunity?.title || 'Application'}</h2>
              <span className={`badge badge-status badge-${app.current_status}`}>
                {formatStatus(app.current_status)}
              </span>
            </div>
            <p className="org-name">{app.opportunity?.organization_name}</p>
            <p className="meta">
              Applied: {formatDate(app.date_applied)} · Updated:{' '}
              {formatDate(app.updated_at)}
            </p>
          </Link>
        ))}
      </div>
    </div>
  )
}
