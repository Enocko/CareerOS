import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { RecommendedOpportunity } from '../types'
import { formatCategory, formatDate, formatVerificationLabel, formatMatchScoreLabel } from '../utils/format'

const RECOMMENDED_PER_PAGE = 10

export function RecommendedPage() {
  const [items, setItems] = useState<RecommendedOpportunity[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [totalPages, setTotalPages] = useState(0)
  const [profileComplete, setProfileComplete] = useState(true)
  const [savingId, setSavingId] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    async function load() {
      setLoading(true)
      setError('')
      setItems([])
      try {
        const resp = await api.listRecommendedOpportunities({
          page: String(page),
          per_page: String(RECOMMENDED_PER_PAGE),
        })
        if (!cancelled) {
          setItems(resp.data)
          setTotalPages(resp.pagination.total_pages)
          setProfileComplete(resp.meta.profile_complete)
          void api.recordRecommendationEvent('recommendation_impression')
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to load recommendations')
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    load()
    return () => {
      cancelled = true
    }
  }, [page])

  async function handleSave(id: string) {
    setSavingId(id)
    try {
      await api.saveOpportunity(id)
      await api.recordRecommendationEvent('opportunity_saved_from_recommendation', id)
      setItems((prev) =>
        prev.map((item) =>
          item.opportunity.id === id
            ? { ...item, opportunity: { ...item.opportunity, is_saved: true } }
            : item,
        ),
      )
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save')
    } finally {
      setSavingId(null)
    }
  }

  function openDetail(id: string) {
    void api.recordRecommendationEvent('recommendation_opened', id)
  }

  return (
    <div>
      <div className="page-header">
        <h1>Recommended for You</h1>
        {!profileComplete && (
          <p className="profile-nudge">
            Complete your <Link to="/profile">profile</Link> for better matches.
          </p>
        )}
      </div>

      {error && <div className="alert alert-error">{error}</div>}
      {loading && <div className="loading">Loading recommendations...</div>}

      {!loading && items.length === 0 && (
        <div className="empty-state">No recommendations available right now.</div>
      )}

      <div className="card-list">
        {items.map((item) => {
          const opp = item.opportunity
          const chips = item.match_summary
            ? item.match_summary.split(' · ').filter(Boolean)
            : item.match_reasons.slice(0, 3)
          return (
            <article key={opp.id} className="card opp-card recommended-card">
              <div className="opp-card-header">
                <div>
                  <p className="match-score">
                    {formatMatchScoreLabel(item.match_score)} · {item.match_score}
                  </p>
                  <h2>
                    <Link
                      to={`/browse/${opp.id}`}
                      state={{ from: 'recommended' }}
                      onClick={() => openDetail(opp.id)}
                    >
                      {opp.title}
                    </Link>
                  </h2>
                </div>
                {opp.is_saved && <span className="badge badge-saved">Saved</span>}
              </div>

              <p className="org-name">{opp.organization_name}</p>

              {chips.length > 0 && (
                <p className="match-chips">{chips.join(' · ')}</p>
              )}

              <details className="match-details">
                <summary>Why this matches</summary>
                <ul>
                  {item.match_reasons.map((reason) => (
                    <li key={reason}>{reason}</li>
                  ))}
                </ul>
              </details>

              <div className="tag-row">
                <span className="badge">{formatCategory(opp.category)}</span>
                {opp.experience_level && (
                  <span className="badge badge-muted">{opp.experience_level.replace(/_/g, ' ')}</span>
                )}
                <span className="badge badge-muted">{opp.work_arrangement}</span>
                {opp.location && <span className="badge badge-muted">{opp.location}</span>}
              </div>

              <p className="meta">Deadline: {formatDate(opp.deadline)}</p>
              {opp.verification_status === 'verified' && (
                <p className="verification-note verified">
                  {formatVerificationLabel(
                    opp.verification_status,
                    opp.source_name,
                    opp.last_checked_at,
                  )}
                </p>
              )}

              <div className="card-actions">
                {!opp.is_saved ? (
                  <button
                    type="button"
                    className="btn btn-secondary"
                    disabled={savingId === opp.id}
                    onClick={() => handleSave(opp.id)}
                  >
                    Save
                  </button>
                ) : (
                  <Link to={`/browse/${opp.id}`} className="btn btn-secondary">
                    View saved
                  </Link>
                )}
                <Link
                  to={`/browse/${opp.id}`}
                  state={{ from: 'recommended' }}
                  className="btn btn-primary"
                  onClick={() => openDetail(opp.id)}
                >
                  View & apply
                </Link>
              </div>
            </article>
          )
        })}
      </div>

      {!loading && totalPages > 1 && (
        <nav className="pagination" aria-label="Recommendation pages">
          <button
            type="button"
            className="btn btn-secondary"
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            disabled={page <= 1}
          >
            Previous
          </button>
          <span className="pagination-status">
            Page {page} of {totalPages}
          </span>
          <button
            type="button"
            className="btn btn-secondary"
            onClick={() => setPage((p) => p + 1)}
            disabled={page >= totalPages}
          >
            Next
          </button>
        </nav>
      )}
    </div>
  )
}