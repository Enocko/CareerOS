import { useEffect, useState } from 'react'
import { Link, useLocation, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { Opportunity } from '../types'
import { formatCategory, formatDate, formatVerificationLabel, formatResearchAwardLabel, formatApplicationAvailabilityLabel, formatListingStatusLabel } from '../utils/format'

export function OpportunityDetailPage() {
  const { id } = useParams<{ id: string }>()
  const location = useLocation()
  const fromRecommended = (location.state as { from?: string } | null)?.from === 'recommended'
  const browseBackType = (location.state as { browseType?: string } | null)?.browseType
  const fromBrowse =
    (location.state as { from?: string } | null)?.from === 'browse' ||
    location.pathname.startsWith('/browse/') ||
    location.pathname.startsWith('/opportunities/') ||
    location.pathname.startsWith('/research/')
  const [opp, setOpp] = useState<Opportunity | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [actionMsg, setActionMsg] = useState('')
  const [reportReason, setReportReason] = useState('')
  const [reportNote, setReportNote] = useState('')
  const [reportSubmitting, setReportSubmitting] = useState(false)
  const [reportSent, setReportSent] = useState(false)

  useEffect(() => {
    if (!id) return
    api
      .getOpportunity(id)
      .then(setOpp)
      .catch((err) =>
        setError(err instanceof Error ? err.message : 'Failed to load opportunity'),
      )
      .finally(() => setLoading(false))
  }, [id])

  async function handleSave() {
    if (!opp) return
    setActionMsg('')
    try {
      await api.saveOpportunity(opp.id)
      setOpp({ ...opp, is_saved: true })
      setActionMsg('Opportunity saved.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save')
    }
  }

  async function handleUnsave() {
    if (!opp) return
    setActionMsg('')
    try {
      await api.unsaveOpportunity(opp.id)
      setOpp({ ...opp, is_saved: false })
      setActionMsg('Opportunity removed from saved.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to unsave')
    }
  }

  async function handleReport(e: React.FormEvent) {
    e.preventDefault()
    if (!opp || !reportReason) return
    setReportSubmitting(true)
    setError('')
    try {
      await api.reportOpportunity(opp.id, reportReason, reportNote.trim() || undefined)
      setReportSent(true)
      setReportReason('')
      setReportNote('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to submit report')
    } finally {
      setReportSubmitting(false)
    }
  }

  async function handleTrack() {
    if (!opp) return
    setActionMsg('')
    try {
      await api.createApplication(opp.id)
      setOpp({ ...opp, has_application: true })
      setActionMsg('Now tracking this opportunity. View it anytime under Applications.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start tracking')
    }
  }

  if (loading) return <div className="loading">Loading opportunity...</div>
  if (error && !opp) return <div className="alert alert-error">{error}</div>
  if (!opp) return <div className="empty-state">Opportunity not found.</div>

  const isResearch = opp.opportunity_type === 'research'
  const researchMeta = (opp.type_metadata ?? {}) as import('../types').ResearchTypeMetadata
  const isClosed = opp.status === 'closed'
  const isStale = opp.verification_status === 'stale'
  const listingWarning = formatListingStatusLabel(opp.status, opp.verification_status)
  const actionable = !isClosed && !isStale
  const researchApplyOpen =
    actionable &&
    isResearch &&
    researchMeta.application_status === 'open' &&
    Boolean(opp.application_url)

  const backHref = fromRecommended
    ? '/recommended'
    : fromBrowse
      ? browseBackType
        ? `/browse?type=${browseBackType}`
        : '/browse'
      : '/browse'
  const backLabel = fromRecommended
    ? 'recommendations'
    : fromBrowse && browseBackType === 'research'
      ? 'research programs'
      : 'opportunities'

  return (
    <div>
      <Link to={backHref} className="back-link">
        ← Back to {backLabel}
      </Link>

      <div className="page-header">
        <h1>{opp.title}</h1>
        <p className="subtitle">{opp.organization_name}</p>
        {opp.verification_status === 'verified' && isResearch && (
          <p className="verification-note verified">
            {formatResearchAwardLabel(opp.source.name, opp.last_checked_at)}
          </p>
        )}
        {opp.verification_status === 'verified' && !isResearch && (
          <p className="verification-note verified">
            {formatVerificationLabel(
              opp.verification_status,
              opp.source.name,
              opp.last_checked_at,
            )}
          </p>
        )}
        {opp.verification_status === 'stale' && !isResearch && (
          <p className="verification-note unverified">
            {formatVerificationLabel(opp.verification_status, opp.source.name, opp.last_checked_at)}
          </p>
        )}
        {opp.verification_status === 'unverified' && (
          <p className="verification-note unverified">
            {formatVerificationLabel(opp.verification_status, opp.source.name, null)}
          </p>
        )}
      </div>

      {error && <div className="alert alert-error">{error}</div>}
      {actionMsg && <div className="alert alert-success">{actionMsg}</div>}
      {listingWarning && (
        <div className="alert alert-warning" role="status">
          {listingWarning}
        </div>
      )}

      <div className="detail-grid">
        <div className="card">
          <div className="tag-row">
            <span className="badge">{formatCategory(isResearch ? 'research' : opp.category)}</span>
            {!isResearch && (
              <span className="badge badge-muted">{opp.work_arrangement}</span>
            )}
            {opp.location && (
              <span className="badge badge-muted">{opp.location}</span>
            )}
            {isResearch && researchMeta.research_area && (
              <span className="badge badge-muted">{researchMeta.research_area}</span>
            )}
          </div>

          <dl className="detail-list">
            {isResearch && (
              <>
                <dt>Application status</dt>
                <dd>
                  {formatApplicationAvailabilityLabel(
                    researchMeta.application_status,
                    researchMeta.application_verification_source_url ?? null,
                    researchMeta.application_verified_at ?? null,
                  )}
                </dd>
              </>
            )}
            <dt>{isResearch ? 'Application deadline' : 'Deadline'}</dt>
            <dd>{formatDate(opp.deadline)}</dd>
            {isResearch ? (
              <>
                {researchMeta.program_start && (
                  <>
                    <dt>Award program start</dt>
                    <dd>{formatDate(researchMeta.program_start)}</dd>
                  </>
                )}
                {researchMeta.program_end && (
                  <>
                    <dt>Award program end</dt>
                    <dd>{formatDate(researchMeta.program_end)}</dd>
                  </>
                )}
                {researchMeta.duration_weeks != null && (
                  <>
                    <dt>Duration</dt>
                    <dd>{researchMeta.duration_weeks} weeks</dd>
                  </>
                )}
                {researchMeta.stipend && (
                  <>
                    <dt>Stipend</dt>
                    <dd>{researchMeta.stipend}</dd>
                  </>
                )}
                {researchMeta.housing_provided != null && (
                  <>
                    <dt>Housing</dt>
                    <dd>{researchMeta.housing_provided ? 'Provided' : 'Not stated'}</dd>
                  </>
                )}
                {researchMeta.travel_support != null && (
                  <>
                    <dt>Travel support</dt>
                    <dd>{researchMeta.travel_support ? 'Available' : 'Not stated'}</dd>
                  </>
                )}
                {researchMeta.citizenship_required && (
                  <>
                    <dt>Citizenship requirement</dt>
                    <dd>{researchMeta.citizenship_required}</dd>
                  </>
                )}
              </>
            ) : (
              <>
                <dt>Start date</dt>
                <dd>{formatDate(opp.start_date)}</dd>
                {opp.compensation && (
                  <>
                    <dt>Compensation</dt>
                    <dd>{opp.compensation}</dd>
                  </>
                )}
              </>
            )}
          </dl>

          <h3>Description</h3>
          <p className="description">{opp.description}</p>

          {opp.eligibility && (
            <>
              <h3>Eligibility (from official source)</h3>
              <p>{opp.eligibility}</p>
            </>
          )}

          {isResearch && !opp.eligibility && (
            <p className="meta">
              This listing reflects an NSF-funded REU site award. CareerOS has not verified whether
              student applications are currently open. Review the official program page or NSF award
              for source-stated requirements. CareerOS does not determine your eligibility.
            </p>
          )}

          {!isResearch && opp.skills.length > 0 && (
            <>
              <h3>Skills</h3>
              <div className="tag-row">
                {opp.skills.map((s) => (
                  <span key={s} className="badge badge-muted">
                    {s}
                  </span>
                ))}
              </div>
            </>
          )}
        </div>

        <div className="card actions-card">
          <h3>Actions</h3>
          <div className="action-buttons">
            {opp.is_saved ? (
              <button type="button" className="btn btn-secondary btn-full" onClick={handleUnsave}>
                Unsave
              </button>
            ) : (
              <button type="button" className="btn btn-secondary btn-full" onClick={handleSave}>
                Save opportunity
              </button>
            )}

            {opp.has_application ? (
              <Link to="/applications" className="btn btn-primary btn-full">
                View application
              </Link>
            ) : !isResearch && actionable ? (
              <button type="button" className="btn btn-primary btn-full" onClick={handleTrack}>
                Start tracking
              </button>
            ) : null}

            {opp.application_url && actionable && (!isResearch || researchApplyOpen) && (
              <a
                href={opp.application_url}
                target="_blank"
                rel="noopener noreferrer"
                className="btn btn-outline btn-full"
                onClick={() => {
                  if (fromRecommended) {
                    void api.recordRecommendationEvent(
                      'official_application_link_clicked',
                      opp.id,
                    )
                  }
                }}
              >
                {isResearch ? 'Apply at official source ↗' : 'Apply on official site ↗'}
              </a>
            )}
            {isResearch && !researchApplyOpen && researchMeta.program_url && (
              <a
                href={researchMeta.program_url}
                target="_blank"
                rel="noopener noreferrer"
                className="btn btn-outline btn-full"
              >
                View program website ↗
              </a>
            )}
            {opp.source.source_url && (
              <a
                href={opp.source.source_url}
                target="_blank"
                rel="noopener noreferrer"
                className="text-link"
              >
                {isResearch ? 'View NSF award record' : `View listing on ${opp.source.name}`}
              </a>
            )}
          </div>

          <details className="report-issue">
            <summary>Report an issue</summary>
            {reportSent ? (
              <p className="meta">Thanks — your report was recorded for review.</p>
            ) : (
              <form onSubmit={handleReport} className="report-form">
                <label>
                  Reason
                  <select
                    value={reportReason}
                    onChange={(e) => setReportReason(e.target.value)}
                    required
                  >
                    <option value="">Select a reason</option>
                    <option value="appears_closed">Opportunity appears closed</option>
                    <option value="broken_link">Application link is broken</option>
                    <option value="incorrect_deadline">Deadline appears incorrect</option>
                    <option value="duplicate">Duplicate opportunity</option>
                    <option value="incorrect_info">Incorrect information</option>
                    <option value="other">Other</option>
                  </select>
                </label>
                <label>
                  Note (optional)
                  <textarea
                    value={reportNote}
                    onChange={(e) => setReportNote(e.target.value)}
                    maxLength={500}
                    rows={3}
                    placeholder="Short details for our review team"
                  />
                </label>
                <button
                  type="submit"
                  className="btn btn-secondary btn-full"
                  disabled={reportSubmitting || !reportReason}
                >
                  {reportSubmitting ? 'Submitting...' : 'Submit report'}
                </button>
              </form>
            )}
          </details>
        </div>
      </div>
    </div>
  )
}
