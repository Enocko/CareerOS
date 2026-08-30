export function formatDate(dateStr: string | null | undefined): string {
  if (!dateStr) return '—'
  const date = new Date(dateStr)
  return date.toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

export function formatStatus(status: string): string {
  return status.replace(/_/g, ' ')
}

export function formatCategory(category: string): string {
  return category.replace(/_/g, ' ')
}

export function formatVerificationLabel(
  status: string,
  sourceName: string,
  lastCheckedAt: string | null | undefined,
): string {
  if (status === 'verified' && sourceName) {
    const checked = lastCheckedAt ? formatRelativeTime(lastCheckedAt) : null
    return checked
      ? `Last verified from ${sourceName} · checked ${checked}`
      : `Last verified from ${sourceName}`
  }
  if (status === 'stale') {
    const checked = lastCheckedAt ? formatRelativeTime(lastCheckedAt) : null
    return checked
      ? `Not recently confirmed at source · last checked ${checked}`
      : 'Not recently confirmed at source — verify before applying'
  }
  if (status === 'unverified') {
    return 'Not source-verified — confirm details before applying'
  }
  return ''
}

export function formatListingStatusLabel(status: string, verificationStatus: string): string | null {
  if (status === 'closed') return 'This opportunity is closed'
  if (verificationStatus === 'stale') return 'Listing may be outdated'
  return null
}

export function formatMatchScoreLabel(score: number): string {
  if (score >= 80) return 'Strong match'
  if (score >= 60) return 'Good match'
  return 'Match score'
}

export function formatReportReason(reason: string): string {
  switch (reason) {
    case 'appears_closed':
      return 'Appears closed'
    case 'broken_link':
      return 'Broken link'
    case 'incorrect_deadline':
      return 'Incorrect deadline'
    case 'duplicate':
      return 'Duplicate'
    case 'incorrect_info':
      return 'Incorrect information'
    default:
      return 'Other'
  }
}

function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr)
  const diffMs = Date.now() - date.getTime()
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60))
  if (diffHours < 1) return 'recently'
  if (diffHours < 24) return `${diffHours}h ago`
  const diffDays = Math.floor(diffHours / 24)
  if (diffDays === 1) return '1 day ago'
  return `${diffDays} days ago`
}

export function formatApplicationStatusLabel(status: string | undefined): string {
  switch (status) {
    case 'open':
      return 'Applications open'
    case 'upcoming':
      return 'Applications upcoming'
    case 'closed':
      return 'Applications closed'
    default:
      return 'Application status unknown'
  }
}

export function formatResearchAwardLabel(
  sourceName: string,
  lastCheckedAt: string | null | undefined,
): string {
  const checked = lastCheckedAt ? formatRelativeTime(lastCheckedAt) : null
  if (checked) {
    return `NSF award verified via ${sourceName} · checked ${checked}`
  }
  return `NSF award verified via ${sourceName}`
}

export function formatApplicationAvailabilityLabel(
  status: string | undefined,
  sourceURL: string | null | undefined,
  verifiedAt: string | null | undefined,
): string {
  if (status === 'open' && sourceURL) {
    const checked = verifiedAt ? formatRelativeTime(verifiedAt) : null
    return checked
      ? `Applications verified open from official source · checked ${checked}`
      : 'Applications verified open from official source'
  }
  return formatApplicationStatusLabel(status)
}
