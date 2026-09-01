import { Link } from 'react-router-dom'
import type { OpportunitySummary, ResearchTypeMetadata } from '../types'
import {
  formatApplicationStatusLabel,
  formatCategory,
  formatDate,
  formatResearchAwardLabel,
  formatVerificationLabel,
} from '../utils/format'

interface OpportunityBrowseCardProps {
  opp: OpportunitySummary
  browseType: string
}

function researchMeta(opp: OpportunitySummary): ResearchTypeMetadata {
  return (opp.type_metadata ?? {}) as ResearchTypeMetadata
}

function isResearchOpportunity(opp: OpportunitySummary): boolean {
  return opp.opportunity_type === 'research' || opp.category === 'research'
}

export function OpportunityBrowseCard({ opp, browseType }: OpportunityBrowseCardProps) {
  const isResearch = isResearchOpportunity(opp)
  const meta = researchMeta(opp)
  const detailPath = `/browse/${opp.id}`
  const fromState = { from: 'browse', browseType: isResearch ? 'research' : browseType }

  if (browseType === 'research' && !isResearch) {
    return null
  }
  if (browseType === 'employment' && isResearch) {
    return null
  }

  if (isResearch) {
    return (
      <Link
        to={detailPath}
        state={fromState}
        className="card opp-card opp-card-research"
      >
        <div className="opp-card-header">
          <h2>{opp.title}</h2>
          {opp.is_saved && <span className="badge badge-saved">Saved</span>}
        </div>
        <p className="org-name">{opp.organization_name}</p>
        <div className="tag-row">
          <span className="badge">Research</span>
          <span className="badge badge-muted">NSF-funded REU</span>
          <span className="badge badge-muted">
            {formatApplicationStatusLabel(meta.application_status)}
          </span>
          {meta.cycle_label && (
            <span className="badge badge-muted">{meta.cycle_label}</span>
          )}
          {meta.research_area && (
            <span className="badge badge-muted">{meta.research_area}</span>
          )}
        </div>
        {opp.deadline && meta.application_status === 'open' && (
          <p className="meta">Application deadline: {formatDate(opp.deadline)}</p>
        )}
        {meta.opens_at && meta.application_status === 'upcoming' && (
          <p className="meta">Expected to open: {formatDate(meta.opens_at)}</p>
        )}
        {opp.verification_status === 'verified' && (
          <p className="verification-note verified">
            {formatResearchAwardLabel(opp.source_name, opp.last_checked_at)}
          </p>
        )}
        {opp.verification_status === 'stale' && (
          <p className="verification-note unverified">
            {formatVerificationLabel(opp.verification_status, opp.source_name, opp.last_checked_at)}
          </p>
        )}
      </Link>
    )
  }

  return (
    <Link to={detailPath} state={fromState} className="card opp-card">
      <div className="opp-card-header">
        <h2>{opp.title}</h2>
        {opp.is_saved && <span className="badge badge-saved">Saved</span>}
      </div>
      <p className="org-name">{opp.organization_name}</p>
      <div className="tag-row">
        <span className="badge">{formatCategory(opp.category)}</span>
        {opp.experience_level && (
          <span className="badge badge-muted">{formatCategory(opp.experience_level)}</span>
        )}
        {opp.career_family && (
          <span className="badge badge-muted">{formatCategory(opp.career_family)}</span>
        )}
        <span className="badge badge-muted">{opp.work_arrangement}</span>
        {opp.listing_count && opp.listing_count > 1 ? (
          <span className="badge badge-muted">
            {opp.listing_count} locations
          </span>
        ) : (
          opp.location && <span className="badge badge-muted">{opp.location}</span>
        )}
      </div>
      <p className="meta">Deadline: {formatDate(opp.deadline)}</p>
      {opp.verification_status === 'verified' && (
        <p className="verification-note verified">
          {formatVerificationLabel(opp.verification_status, opp.source_name, opp.last_checked_at)}
        </p>
      )}
      {opp.verification_status === 'stale' && (
        <p className="verification-note unverified">
          {formatVerificationLabel(opp.verification_status, opp.source_name, opp.last_checked_at)}
        </p>
      )}
    </Link>
  )
}
