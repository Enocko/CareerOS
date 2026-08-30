import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { Application } from '../types'
import { APPLICATION_STATUSES } from '../types'
import { formatDate, formatStatus } from '../utils/format'

const REMOVABLE_STATUSES = new Set(['saved', 'preparing'])

export function ApplicationDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [app, setApp] = useState<Application | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [saving, setSaving] = useState(false)
  const [removing, setRemoving] = useState(false)
  const [status, setStatus] = useState('')
  const [notes, setNotes] = useState('')

  useEffect(() => {
    if (!id) return
    api
      .getApplication(id)
      .then((data) => {
        setApp(data)
        setStatus(data.current_status)
        setNotes(data.notes || '')
      })
      .catch((err) =>
        setError(err instanceof Error ? err.message : 'Failed to load application'),
      )
      .finally(() => setLoading(false))
  }, [id])

  async function handleUpdate() {
    if (!app) return
    setError('')
    setSuccess('')
    setSaving(true)
    try {
      await api.updateApplication(app.id, {
        current_status: status,
        notes: notes || null,
      })
      const full = await api.getApplication(app.id)
      setApp(full)
      setSuccess('Application updated.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update application')
    } finally {
      setSaving(false)
    }
  }

  async function handleRemove() {
    if (!app) return
    const confirmed = window.confirm(
      'Remove this opportunity from your Applications dashboard? You have not applied yet, so this will delete the tracking record.',
    )
    if (!confirmed) return

    setError('')
    setSuccess('')
    setRemoving(true)
    try {
      await api.deleteApplication(app.id)
      navigate('/applications')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to remove tracking')
      setRemoving(false)
    }
  }

  if (loading) return <div className="loading">Loading application...</div>
  if (error && !app) return <div className="alert alert-error">{error}</div>
  if (!app) return <div className="empty-state">Application not found.</div>

  const canRemove = REMOVABLE_STATUSES.has(app.current_status)

  return (
    <div>
      <Link to="/applications" className="back-link">
        ← Back to dashboard
      </Link>

      <div className="page-header">
        <h1>{app.opportunity?.title || 'Application'}</h1>
        <p className="subtitle">{app.opportunity?.organization_name}</p>
        {app.current_status === 'saved' && (
          <p className="meta">
            Status &quot;saved&quot; means you are tracking this opportunity — not that you have
            already applied.
          </p>
        )}
      </div>

      {error && <div className="alert alert-error">{error}</div>}
      {success && <div className="alert alert-success">{success}</div>}

      <div className="detail-grid">
        <div className="card">
          <h3>Update application</h3>
          <div className="form">
            <label>
              Status
              <select value={status} onChange={(e) => setStatus(e.target.value)}>
                {APPLICATION_STATUSES.map((s) => (
                  <option key={s} value={s}>
                    {formatStatus(s)}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Notes
              <textarea
                rows={4}
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
              />
            </label>
            <button
              type="button"
              className="btn btn-primary"
              onClick={handleUpdate}
              disabled={saving}
            >
              {saving ? 'Saving...' : 'Update application'}
            </button>
          </div>

          {canRemove && (
            <button
              type="button"
              className="btn btn-secondary btn-full"
              style={{ marginTop: '1rem' }}
              onClick={handleRemove}
              disabled={removing}
            >
              {removing ? 'Removing...' : 'Remove from tracking'}
            </button>
          )}

          <dl className="detail-list" style={{ marginTop: '1.5rem' }}>
            <dt>Date applied</dt>
            <dd>{formatDate(app.date_applied)}</dd>
            <dt>Created</dt>
            <dd>{formatDate(app.created_at)}</dd>
          </dl>

          {app.opportunity?.application_url && (
            <a
              href={app.opportunity.application_url}
              target="_blank"
              rel="noopener noreferrer"
              className="btn btn-outline"
              style={{ marginTop: '1rem' }}
            >
              Open application link ↗
            </a>
          )}
        </div>

        <div className="card">
          <h3>Status history</h3>
          {app.status_history && app.status_history.length > 0 ? (
            <ul className="history-list">
              {app.status_history.map((entry) => (
                <li key={entry.id}>
                  <span className="history-transition">
                    {entry.from_status
                      ? `${formatStatus(entry.from_status)} → `
                      : 'Created → '}
                    <strong>{formatStatus(entry.to_status)}</strong>
                  </span>
                  <span className="history-date">
                    {formatDate(entry.changed_at)}
                  </span>
                </li>
              ))}
            </ul>
          ) : (
            <p className="meta">No status history yet.</p>
          )}
        </div>
      </div>
    </div>
  )
}
