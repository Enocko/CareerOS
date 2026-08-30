import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'

interface QueueItem {
  id: string
  title: string
  organization_name: string
  source_url?: string | null
  program_url?: string | null
  application_status: string
  priority_score: number
}

export function ResearchVerificationPage() {
  const [queue, setQueue] = useState<QueueItem[]>([])
  const [selected, setSelected] = useState<QueueItem | null>(null)
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')
  const [loading, setLoading] = useState(true)
  const [form, setForm] = useState({
    application_status: 'unknown',
    verification_source_url: '',
    application_url: '',
    deadline: '',
    opens_at: '',
    cycle_label: '',
    notes: '',
  })

  useEffect(() => {
    api
      .getResearchVerificationQueue()
      .then((resp) => setQueue(resp.data))
      .catch((err) =>
        setError(err instanceof Error ? err.message : 'Failed to load verification queue'),
      )
      .finally(() => setLoading(false))
  }, [])

  function selectItem(item: QueueItem) {
    setSelected(item)
    setMessage('')
    setForm({
      application_status: item.application_status || 'unknown',
      verification_source_url: item.program_url || item.source_url || '',
      application_url: '',
      deadline: '',
      opens_at: '',
      cycle_label: '',
      notes: '',
    })
  }

  async function submitVerification(e: React.FormEvent) {
    e.preventDefault()
    if (!selected) return
    setError('')
    setMessage('')
    try {
      await api.submitResearchVerification(selected.id, {
        application_status: form.application_status,
        verification_source_url: form.verification_source_url,
        verification_method: 'manual_official_page',
        application_url: form.application_url || null,
        deadline: form.deadline || null,
        opens_at: form.opens_at || null,
        cycle_label: form.cycle_label || null,
        notes: form.notes || null,
      })
      setMessage('Verification saved.')
      const resp = await api.getResearchVerificationQueue()
      setQueue(resp.data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Verification failed')
    }
  }

  if (loading) return <div className="loading">Loading verification queue...</div>

  return (
    <div>
      <div className="page-header">
        <h1>Research verification</h1>
        <p className="subtitle">Internal workflow for NSF REU application availability.</p>
        <Link to="/browse?type=research" className="back-link">
          ← Back to Research
        </Link>
      </div>

      {error && <div className="alert alert-error">{error}</div>}
      {message && <div className="alert alert-success">{message}</div>}

      <div className="detail-grid">
        <div className="card">
          <h2>Priority queue</h2>
          <ul className="admin-queue">
            {queue.map((item) => (
              <li key={item.id}>
                <button type="button" className="text-link" onClick={() => selectItem(item)}>
                  [{item.priority_score}] {item.organization_name} — {item.title}
                </button>
              </li>
            ))}
          </ul>
        </div>

        {selected && (
          <form className="card" onSubmit={submitVerification}>
            <h2>Verify availability</h2>
            <p>
              <strong>{selected.title}</strong>
              <br />
              {selected.organization_name}
            </p>
            {selected.program_url && (
              <p>
                <a href={selected.program_url} target="_blank" rel="noopener noreferrer">
                  Program page ↗
                </a>
              </p>
            )}
            {selected.source_url && (
              <p>
                <a href={selected.source_url} target="_blank" rel="noopener noreferrer">
                  NSF award ↗
                </a>
              </p>
            )}

            <label>
              Application status
              <select
                value={form.application_status}
                onChange={(e) => setForm({ ...form, application_status: e.target.value })}
              >
                <option value="open">open</option>
                <option value="upcoming">upcoming</option>
                <option value="closed">closed</option>
                <option value="unknown">unknown</option>
              </select>
            </label>
            <label>
              Verification source URL
              <input
                value={form.verification_source_url}
                onChange={(e) => setForm({ ...form, verification_source_url: e.target.value })}
                required={form.application_status !== 'unknown'}
              />
            </label>
            <label>
              Application URL (required for open)
              <input
                value={form.application_url}
                onChange={(e) => setForm({ ...form, application_url: e.target.value })}
              />
            </label>
            <label>
              Opens at (YYYY-MM-DD)
              <input
                value={form.opens_at}
                onChange={(e) => setForm({ ...form, opens_at: e.target.value })}
              />
            </label>
            <label>
              Deadline (YYYY-MM-DD)
              <input
                value={form.deadline}
                onChange={(e) => setForm({ ...form, deadline: e.target.value })}
              />
            </label>
            <label>
              Cycle label
              <input
                value={form.cycle_label}
                onChange={(e) => setForm({ ...form, cycle_label: e.target.value })}
                placeholder="Summer 2027"
              />
            </label>
            <label>
              Notes
              <textarea
                value={form.notes}
                onChange={(e) => setForm({ ...form, notes: e.target.value })}
              />
            </label>
            <button type="submit" className="btn btn-primary">
              Save verification
            </button>
          </form>
        )}
      </div>
    </div>
  )
}
