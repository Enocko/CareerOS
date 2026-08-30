import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { AdminOverview, OpportunityReportQueueItem } from '../types'
import { formatDate, formatReportReason } from '../utils/format'

export function AdminPage() {
  const [overview, setOverview] = useState<AdminOverview | null>(null)
  const [reports, setReports] = useState<OpportunityReportQueueItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [resolvingId, setResolvingId] = useState<string | null>(null)

  async function load() {
    setLoading(true)
    setError('')
    try {
      const [overviewResp, reportsResp] = await Promise.all([
        api.getAdminOverview(),
        api.listOpportunityReports(),
      ])
      setOverview(overviewResp)
      setReports(reportsResp.data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load admin overview')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  async function resolveReport(opportunityId: string, status: 'resolved' | 'dismissed') {
    setResolvingId(opportunityId)
    try {
      await api.resolveOpportunityReports(opportunityId, status)
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update report')
    } finally {
      setResolvingId(null)
    }
  }

  return (
    <div>
      <div className="page-header">
        <h1>Operator Console</h1>
        <p className="subtitle">Catalog health, reports, and verification queues.</p>
      </div>

      <div className="admin-links card">
        <Link to="/admin/research" className="text-link">
          Research verification queue →
        </Link>
      </div>

      {error && <div className="alert alert-error">{error}</div>}
      {loading && <div className="loading">Loading operator data...</div>}

      {overview && (
        <div className="card admin-metrics">
          <h2>Catalog snapshot</h2>
          <ul className="admin-metric-list">
            <li>Employment visible: {overview.catalog_metrics.employment_visible}</li>
            <li>Research programs: {overview.catalog_metrics.research_candidate}</li>
            <li>Research open / upcoming / unknown: {overview.catalog_metrics.research_open} / {overview.catalog_metrics.research_upcoming} / {overview.catalog_metrics.research_unknown}</li>
            <li>Stale listings: {overview.catalog_metrics.stale_listings}</li>
            <li>Pending student reports: {overview.pending_reports}</li>
            <li>Duplicate groups: {overview.duplicate_groups} ({overview.excess_duplicate_records} excess records)</li>
            <li>Failed background jobs: {overview.job_metrics.failed}</li>
          </ul>
        </div>
      )}

      {overview && overview.source_health.length > 0 && (
        <div className="card">
          <h2>Source health</h2>
          <div className="table-scroll">
            <table className="admin-table">
              <thead>
                <tr>
                  <th>Source</th>
                  <th>Adapter</th>
                  <th>Last run</th>
                  <th>Open</th>
                  <th>Fails</th>
                </tr>
              </thead>
              <tbody>
                {overview.source_health.map((row) => (
                  <tr key={row.source_name}>
                    <td>{row.source_name}</td>
                    <td>{row.adapter}</td>
                    <td>{row.last_run_status || '—'}</td>
                    <td>{row.verified_open_count}</td>
                    <td>{row.consecutive_fails}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <div className="card">
        <h2>Pending opportunity reports</h2>
        {reports.length === 0 ? (
          <p className="meta">No pending reports.</p>
        ) : (
          <div className="card-list">
            {reports.map((item) => (
              <article key={item.opportunity_id} className="opp-card">
                <h3>
                  <Link to={`/browse/${item.opportunity_id}`}>{item.opportunity_title}</Link>
                </h3>
                <p className="org-name">{item.organization_name}</p>
                <p className="meta">
                  {item.source_name} · {item.report_count} report(s) · latest{' '}
                  {formatDate(item.latest_report_at)}
                </p>
                <p className="meta">
                  Reasons: {item.reasons.map(formatReportReason).join(', ')}
                </p>
                {item.source_url && (
                  <a href={item.source_url} target="_blank" rel="noopener noreferrer" className="text-link">
                    Official source ↗
                  </a>
                )}
                <div className="card-actions">
                  <button
                    type="button"
                    className="btn btn-primary"
                    disabled={resolvingId === item.opportunity_id}
                    onClick={() => resolveReport(item.opportunity_id, 'resolved')}
                  >
                    Mark resolved
                  </button>
                  <button
                    type="button"
                    className="btn btn-secondary"
                    disabled={resolvingId === item.opportunity_id}
                    onClick={() => resolveReport(item.opportunity_id, 'dismissed')}
                  >
                    Dismiss
                  </button>
                </div>
              </article>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
