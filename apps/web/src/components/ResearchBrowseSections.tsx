import { OpportunityBrowseCard } from './OpportunityBrowseCard'
import type { OpportunitySummary } from '../types'

interface ResearchBrowseSectionsProps {
  opportunities: OpportunitySummary[]
  browseType: string
}

type ResearchSection = {
  key: string
  title: string
  description?: string
  className?: string
  items: OpportunitySummary[]
}

function sectionForStatus(status: string | undefined): string {
  switch (status) {
    case 'open':
      return 'open'
    case 'upcoming':
      return 'upcoming'
    case 'closed':
      return 'closed'
    default:
      return 'unknown'
  }
}

function buildSections(opportunities: OpportunitySummary[]): ResearchSection[] {
  const buckets: Record<string, OpportunitySummary[]> = {
    open: [],
    upcoming: [],
    unknown: [],
    closed: [],
  }

  for (const opp of opportunities) {
    const meta = (opp.type_metadata ?? {}) as { application_status?: string }
    buckets[sectionForStatus(meta.application_status)].push(opp)
  }

  const sections: ResearchSection[] = []
  if (buckets.open.length > 0) {
    sections.push({
      key: 'open',
      title: 'Applications open now',
      items: buckets.open,
    })
  }
  if (buckets.upcoming.length > 0) {
    sections.push({
      key: 'upcoming',
      title: 'Applications upcoming',
      items: buckets.upcoming,
    })
  }
  if (buckets.unknown.length > 0) {
    sections.push({
      key: 'unknown',
      title: 'Availability not yet verified',
      description:
        'NSF-funded REU programs. CareerOS has not verified whether student applications are currently open.',
      className: 'research-section-unknown',
      items: buckets.unknown,
    })
  }
  if (buckets.closed.length > 0) {
    sections.push({
      key: 'closed',
      title: 'Applications closed',
      items: buckets.closed,
    })
  }
  return sections
}

export function ResearchBrowseSections({
  opportunities,
  browseType,
}: ResearchBrowseSectionsProps) {
  const sections = buildSections(opportunities)

  return (
    <div className="research-browse-sections">
      {sections.map((section) => (
        <section
          key={section.key}
          className={`research-section${section.className ? ` ${section.className}` : ''}`}
        >
          <h2 className="research-section-title">{section.title}</h2>
          {section.description && (
            <p className="research-section-desc">{section.description}</p>
          )}
          <div className="card-list">
            {section.items.map((opp) => (
              <OpportunityBrowseCard key={opp.id} opp={opp} browseType={browseType} />
            ))}
          </div>
        </section>
      ))}
    </div>
  )
}
