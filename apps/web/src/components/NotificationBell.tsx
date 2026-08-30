import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'

export function NotificationBell() {
  const [unread, setUnread] = useState(0)

  useEffect(() => {
    let cancelled = false
    async function load() {
      try {
        const resp = await api.getUnreadNotificationCount()
        if (!cancelled) setUnread(resp.unread)
      } catch {
        // ignore when logged out or API unavailable
      }
    }
    load()
    const timer = window.setInterval(load, 60_000)
    return () => {
      cancelled = true
      window.clearInterval(timer)
    }
  }, [])

  return (
    <Link to="/notifications" className="notification-bell" aria-label={`Notifications${unread ? `, ${unread} unread` : ''}`}>
      🔔
      {unread > 0 && <span className="notification-badge">{unread > 9 ? '9+' : unread}</span>}
    </Link>
  )
}
