import { useEffect, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { api } from '../api/client'
import { OpportunityBrowseCard } from '../components/OpportunityBrowseCard'
import { ResearchBrowseSections } from '../components/ResearchBrowseSections'
import type { OpportunitySummary } from '../types'

export type BrowseType = 'all' | 'employment' | 'research'

function parseBrowseType(raw: string | null): BrowseType {
  if (raw === 'employment' || raw === 'research') return raw
  return 'all'
}

function useDebouncedValue<T>(value: T, delayMs: number): T {
  const [debounced, setDebounced] = useState(value)

  useEffect(() => {
    const timer = window.setTimeout(() => setDebounced(value), delayMs)
    return () => window.clearTimeout(timer)
  }, [value, delayMs])

  return debounced
}

const OPPORTUNITIES_PER_PAGE = 20

const TYPE_OPTIONS: { value: BrowseType; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'employment', label: 'Internships & Jobs' },
  { value: 'research', label: 'Research' },
]

function isResearchOpportunity(opp: OpportunitySummary): boolean {
  return opp.opportunity_type === 'research' || opp.category === 'research'
}

function isEmploymentOpportunity(opp: OpportunitySummary): boolean {
  return opp.opportunity_type === 'employment' || (!isResearchOpportunity(opp) && opp.category !== 'research')
}

export function OpportunitiesPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const browseType = parseBrowseType(searchParams.get('type'))
  const category = searchParams.get('category') ?? ''
  const workArrangement = searchParams.get('work_arrangement') ?? ''

  const initialQuery = searchParams.get('q') ?? ''
  const [searchInput, setSearchInput] = useState(initialQuery)
  const [page, setPage] = useState(() => Math.max(1, Number(searchParams.get('page') ?? '1') || 1))
  const debouncedSearch = useDebouncedValue(searchInput, 300)

  const [opportunities, setOpportunities] = useState<OpportunitySummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [totalPages, setTotalPages] = useState(0)

  const prevBrowseType = useRef(browseType)

  useEffect(() => {
    if (!searchParams.get('type')) {
      const next = new URLSearchParams(searchParams)
      next.set('type', 'all')
      setSearchParams(next, { replace: true })
    }
  }, [searchParams, setSearchParams])

  // Keep URL in sync for bookmarking / refresh (do not drive fetch from URL q).
  useEffect(() => {
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev)
        next.set('type', browseType)
        const q = debouncedSearch.trim()
        if (q) next.set('q', q)
        else next.delete('q')
        if (category && browseType !== 'research') next.set('category', category)
        else next.delete('category')
        if (workArrangement && browseType !== 'research') next.set('work_arrangement', workArrangement)
        else next.delete('work_arrangement')
        if (page > 1) next.set('page', String(page))
        else next.delete('page')
        return next
      },
      { replace: true },
    )
  }, [browseType, debouncedSearch, category, workArrangement, page, setSearchParams])

  useEffect(() => {
    if (prevBrowseType.current !== browseType) {
      prevBrowseType.current = browseType
      setPage(1)
    }
  }, [browseType])

  useEffect(() => {
    setPage(1)
  }, [debouncedSearch, category, workArrangement])

  useEffect(() => {
    let cancelled = false

    async function loadOpportunities() {
      setLoading(true)
      setError('')
      setOpportunities([])
      try {
        const filters: Record<string, string> = {
          type: browseType,
          page: String(page),
          per_page: String(OPPORTUNITIES_PER_PAGE),
        }
        if (browseType === 'employment' || browseType === 'research') {
          filters.opportunity_type = browseType
        }
        const q = debouncedSearch.trim()
        if (q) filters.q = q
        if (category && browseType !== 'research') filters.category = category
        if (workArrangement && browseType !== 'research') {
          filters.work_arrangement = workArrangement
        }

        const resp = await api.listOpportunities(filters)
        if (cancelled) return

        let data = resp.data
        if (browseType === 'research') {
          data = data.filter(isResearchOpportunity)
        } else if (browseType === 'employment') {
          data = data.filter(isEmploymentOpportunity)
        }

        setOpportunities(data)
        setTotalPages(resp.pagination.total_pages)
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to load opportunities')
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    loadOpportunities()
    return () => {
      cancelled = true
    }
  }, [browseType, debouncedSearch, category, workArrangement, page])

  function patchParams(patch: Record<string, string | null>) {
    const next = new URLSearchParams(searchParams)
    next.set('type', browseType)
    for (const [key, value] of Object.entries(patch)) {
      if (value == null || value === '') next.delete(key)
      else next.set(key, value)
    }
    setSearchParams(next)
  }

  function setBrowseType(next: BrowseType) {
    setOpportunities([])
    setLoading(true)
    setPage(1)
    const nextParams = new URLSearchParams(searchParams)
    nextParams.set('type', next)
    nextParams.delete('page')
    if (next === 'research') {
      nextParams.delete('category')
      nextParams.delete('work_arrangement')
    }
    setSearchParams(nextParams)
  }

  function handleSearch(e: React.FormEvent) {
    e.preventDefault()
  }

  const showEmploymentFilters = browseType !== 'research'
  const hasActiveFilters = Boolean(
    debouncedSearch.trim() || (showEmploymentFilters && (category || workArrangement)),
  )

  function clearFilters() {
    setSearchInput('')
    setPage(1)
    patchParams({ q: null, category: null, work_arrangement: null, page: null })
  }

  function goToPage(nextPage: number) {
    setPage(nextPage)
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  function emptyMessage(): string {
    if (hasActiveFilters) return 'No opportunities match your filters.'
    switch (browseType) {
      case 'research':
        return 'No research applications are verified open right now. You can still explore NSF-funded research programs whose current application availability has not yet been verified.'
      case 'employment':
        return 'No internships or jobs found.'
      default:
        return 'No opportunities and programs found.'
    }
  }

  return (
    <div>
      <div className="page-header">
        <h1>Opportunities</h1>
      </div>

      <div className="browse-type-selector" role="tablist" aria-label="Opportunity type">
        {TYPE_OPTIONS.map((opt) => (
          <button
            key={opt.value}
            type="button"
            role="tab"
            aria-selected={browseType === opt.value}
            className={`browse-type-btn${browseType === opt.value ? ' active' : ''}`}
            onClick={() => setBrowseType(opt.value)}
          >
            {opt.label}
          </button>
        ))}
      </div>

      <form onSubmit={handleSearch} className="card filters">
        <input
          type="search"
          aria-label={
            browseType === 'research'
              ? 'Search research programs'
              : browseType === 'employment'
                ? 'Search internships and jobs'
                : 'Search opportunities'
          }
          placeholder={
            browseType === 'research'
              ? 'Search REU sites, universities, research areas...'
              : browseType === 'employment'
                ? 'Search by title, organization...'
                : 'Search opportunities and programs...'
          }
          value={searchInput}
          onChange={(e) => setSearchInput(e.target.value)}
        />
        {showEmploymentFilters && (
          <>
            <select
              value={category}
              onChange={(e) => patchParams({ category: e.target.value || null, page: null })}
              aria-label="Filter by category"
            >
              <option value="">All categories</option>
              <option value="internship">Internship</option>
              <option value="full_time">Full-time</option>
              <option value="fellowship">Fellowship</option>
              <option value="scholarship">Scholarship</option>
              <option value="hackathon">Hackathon</option>
            </select>
            <select
              value={workArrangement}
              onChange={(e) =>
                patchParams({ work_arrangement: e.target.value || null, page: null })
              }
              aria-label="Filter by work arrangement"
            >
              <option value="">All arrangements</option>
              <option value="remote">Remote</option>
              <option value="hybrid">Hybrid</option>
              <option value="on_site">On-site</option>
            </select>
          </>
        )}
        {hasActiveFilters ? (
          <button type="button" className="btn btn-secondary" onClick={clearFilters}>
            Clear
          </button>
        ) : (
          <button type="submit" className="btn btn-primary">
            Search
          </button>
        )}
      </form>

      {error && <div className="alert alert-error">{error}</div>}
      {loading && <div className="loading">Loading opportunities...</div>}

      {!loading && opportunities.length === 0 && (
        <div className="empty-state">{emptyMessage()}</div>
      )}

      {!loading && browseType === 'research' ? (
        <ResearchBrowseSections opportunities={opportunities} browseType={browseType} />
      ) : (
        !loading && (
          <div className="card-list">
            {opportunities.map((opp) => (
              <OpportunityBrowseCard key={opp.id} opp={opp} browseType={browseType} />
            ))}
          </div>
        )
      )}

      {!loading && totalPages > 1 && (
        <nav className="pagination" aria-label="Opportunities pages">
          <button
            type="button"
            className="btn btn-secondary"
            onClick={() => goToPage(page - 1)}
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
            onClick={() => goToPage(page + 1)}
            disabled={page >= totalPages}
          >
            Next
          </button>
        </nav>
      )}
    </div>
  )
}
