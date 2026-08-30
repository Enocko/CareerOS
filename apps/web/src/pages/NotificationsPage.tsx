import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { NotificationItem } from '../types'

export function NotificationsPage() {
  const [items, setItems] = useState<NotificationItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  async function load() {
    setLoading(true)
    setError('')
    try {
      const resp = await api.listNotifications()
      setItems(resp.data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load notifications')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  async function markRead(id: string) {
    await api.markNotificationRead(id)
    setItems((prev) =>
      prev.map((n) => (n.id === id ? { ...n, read_at: new Date().toISOString() } : n)),
    )
  }

  async function markAllRead() {
    await api.markAllNotificationsRead()
    await load()
  }

  return (
    <div>
      <div className="page-header">
        <h1>Notifications</h1>
        <p className="subtitle">Deadline reminders and updates.</p>
      </div>

      {items.some((n) => !n.read_at) && (
        <button type="button" className="btn btn-secondary" onClick={markAllRead}>
          Mark all read
        </button>
      )}

      {error && <div className="alert alert-error">{error}</div>}
      {loading && <div className="loading">Loading notifications...</div>}

      {!loading && items.length === 0 && (
        <div className="empty-state">No notifications yet.</div>
      )}

      <div className="card-list">
        {items.map((n) => (
          <article key={n.id} className={`card notification-card${n.read_at ? '' : ' unread'}`}>
            <h2>{n.title}</h2>
            <p>{n.message}</p>
            <p className="meta">{new Date(n.created_at).toLocaleString()}</p>
            <div className="card-actions">
              {n.opportunity_id && (
                <Link
                  to={
                    n.application_id
                      ? `/applications`
                      : `/browse/${n.opportunity_id}`
                  }
                  className="btn btn-primary"
                  onClick={() => !n.read_at && markRead(n.id)}
                >
                  View
                </Link>
              )}
              {!n.read_at && (
                <button type="button" className="btn btn-secondary" onClick={() => markRead(n.id)}>
                  Mark read
                </button>
              )}
            </div>
          </article>
        ))}
      </div>
    </div>
  )
}
