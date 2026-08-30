import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { OpportunitySummary } from '../types'
import { formatCategory, formatDate, formatListingStatusLabel } from '../utils/format'

export function SavedPage() {
  const [saved, setSaved] = useState<OpportunitySummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [removingId, setRemovingId] = useState<string | null>(null)

  useEffect(() => {
    api
      .listSavedOpportunities()
      .then((resp) => setSaved(resp.data))
      .catch((err) =>
        setError(err instanceof Error ? err.message : 'Failed to load saved opportunities'),
      )
      .finally(() => setLoading(false))
  }, [])

  async function handleUnsave(id: string) {
    setRemovingId(id)
    setError('')
    try {
      await api.unsaveOpportunity(id)
      setSaved((prev) => prev.filter((item) => item.id !== id))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to unsave')
    } finally {
      setRemovingId(null)
    }
  }

  return (
    <div>
      <div className="page-header">
        <h1>Saved Opportunities</h1>
        <p className="subtitle">Opportunities you&apos;ve bookmarked for later.</p>
      </div>

      {error && <div className="alert alert-error">{error}</div>}
      {loading && <div className="loading">Loading saved opportunities...</div>}

      {!loading && saved.length === 0 && (
        <div className="empty-state">
          No saved opportunities yet.{' '}
          <Link to="/browse">Browse opportunities</Link>
        </div>
      )}

      <div className="card-list">
        {saved.map((opp) => {
          const listingNote = formatListingStatusLabel(opp.status, opp.verification_status)
          const isResearch = opp.opportunity_type === 'research'
          return (
            <article key={opp.id} className="card opp-card">
              <div className="opp-card-header">
                <h2>
                  <Link to={`/browse/${opp.id}`}>{opp.title}</Link>
                </h2>
                {listingNote && <span className="badge badge-muted">{listingNote}</span>}
              </div>
              <p className="org-name">{opp.organization_name}</p>
              <div className="tag-row">
                <span className="badge">
                  {formatCategory(isResearch ? 'research' : opp.category)}
                </span>
                {!isResearch && opp.work_arrangement && (
                  <span className="badge badge-muted">{opp.work_arrangement}</span>
                )}
                {opp.has_application && (
                  <span className="badge badge-saved">Tracking</span>
                )}
              </div>
              <p className="meta">Deadline: {formatDate(opp.deadline)}</p>
              <div className="card-actions">
                <Link to={`/browse/${opp.id}`} className="btn btn-primary">
                  View
                </Link>
                <button
                  type="button"
                  className="btn btn-secondary"
                  disabled={removingId === opp.id}
                  onClick={() => handleUnsave(opp.id)}
                >
                  Unsave
                </button>
              </div>
            </article>
          )
        })}
      </div>
    </div>
  )
}
