import { useEffect, useState, type FormEvent } from 'react'
import { api } from '../api/client'
import type { Profile } from '../types'

const emptyProfile: Partial<Profile> = {
  first_name: '',
  last_name: '',
  university: 'Grambling State University',
  major: '',
  graduation_year: undefined,
  career_interests: [],
  desired_roles: [],
  skills: [],
  technologies: [],
  preferred_locations: [],
  work_arrangement: 'remote',
  experience_level: 'intern',
  github_url: '',
  linkedin_url: '',
  portfolio_url: '',
}

function parseList(value: string): string[] {
  return value
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}

function joinList(items: string[] | undefined): string {
  return items?.join(', ') || ''
}

export function ProfilePage() {
  const [form, setForm] = useState<Partial<Profile>>(emptyProfile)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [isNew, setIsNew] = useState(true)

  useEffect(() => {
    api
      .getProfile()
      .then((profile) => {
        setForm(profile)
        setIsNew(false)
      })
      .catch((err) => {
        if (err instanceof Error && err.message.includes('404')) {
          setIsNew(true)
        } else {
          setError(err instanceof Error ? err.message : 'Failed to load profile')
        }
      })
      .finally(() => setLoading(false))
  }, [])

  function updateField<K extends keyof Profile>(key: K, value: Profile[K]) {
    setForm((prev) => ({ ...prev, [key]: value }))
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setSuccess('')
    setSaving(true)

    try {
      const payload = {
        ...form,
        github_url: form.github_url || null,
        linkedin_url: form.linkedin_url || null,
        portfolio_url: form.portfolio_url || null,
        graduation_year: form.graduation_year
          ? Number(form.graduation_year)
          : null,
      }
      const saved = await api.updateProfile(payload)
      setForm(saved)
      setIsNew(false)
      setSuccess('Profile saved successfully.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save profile')
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <div className="loading">Loading profile...</div>

  return (
    <div>
      <div className="page-header">
        <h1>Your Profile</h1>
        <p className="subtitle">
          {isNew
            ? 'Complete your profile to get started.'
            : 'Update your career information.'}
        </p>
      </div>

      {error && <div className="alert alert-error">{error}</div>}
      {success && <div className="alert alert-success">{success}</div>}

      <form onSubmit={handleSubmit} className="card form-grid">
        <div className="form-row">
          <label>
            First name
            <input
              value={form.first_name || ''}
              onChange={(e) => updateField('first_name', e.target.value)}
            />
          </label>
          <label>
            Last name
            <input
              value={form.last_name || ''}
              onChange={(e) => updateField('last_name', e.target.value)}
            />
          </label>
        </div>

        <label>
          University
          <input
            value={form.university || ''}
            onChange={(e) => updateField('university', e.target.value)}
          />
        </label>

        <div className="form-row">
          <label>
            Major
            <input
              value={form.major || ''}
              onChange={(e) => updateField('major', e.target.value)}
            />
          </label>
          <label>
            Graduation year
            <input
              type="number"
              min={2020}
              max={2040}
              value={form.graduation_year || ''}
              onChange={(e) =>
                updateField(
                  'graduation_year',
                  e.target.value ? Number(e.target.value) : null,
                )
              }
            />
          </label>
        </div>

        <div className="form-row">
          <label>
            Work arrangement
            <select
              value={form.work_arrangement || ''}
              onChange={(e) => updateField('work_arrangement', e.target.value)}
            >
              <option value="remote">Remote</option>
              <option value="hybrid">Hybrid</option>
              <option value="on_site">On-site</option>
              <option value="flexible">Flexible</option>
            </select>
          </label>
          <label>
            Experience level
            <select
              value={form.experience_level || ''}
              onChange={(e) => updateField('experience_level', e.target.value)}
            >
              <option value="intern">Intern</option>
              <option value="entry">Entry</option>
              <option value="mid">Mid</option>
              <option value="senior">Senior</option>
            </select>
          </label>
        </div>

        <label>
          Skills (comma-separated)
          <input
            value={joinList(form.skills)}
            onChange={(e) => updateField('skills', parseList(e.target.value))}
          />
        </label>

        <label>
          Technologies (comma-separated)
          <input
            value={joinList(form.technologies)}
            onChange={(e) =>
              updateField('technologies', parseList(e.target.value))
            }
          />
        </label>

        <label>
          Career interests (comma-separated)
          <input
            value={joinList(form.career_interests)}
            onChange={(e) =>
              updateField('career_interests', parseList(e.target.value))
            }
          />
        </label>

        <label>
          Desired roles (comma-separated)
          <input
            value={joinList(form.desired_roles)}
            onChange={(e) =>
              updateField('desired_roles', parseList(e.target.value))
            }
            placeholder="e.g. software engineer, data analyst"
          />
        </label>

        <label>
          Preferred locations (comma-separated)
          <input
            value={joinList(form.preferred_locations)}
            onChange={(e) =>
              updateField('preferred_locations', parseList(e.target.value))
            }
            placeholder="e.g. remote, Dallas TX"
          />
        </label>

        <label>
          GitHub URL
          <input
            type="url"
            value={form.github_url || ''}
            onChange={(e) => updateField('github_url', e.target.value)}
          />
        </label>

        <label>
          LinkedIn URL
          <input
            type="url"
            value={form.linkedin_url || ''}
            onChange={(e) => updateField('linkedin_url', e.target.value)}
          />
        </label>

        <label>
          Portfolio URL
          <input
            type="url"
            value={form.portfolio_url || ''}
            onChange={(e) => updateField('portfolio_url', e.target.value)}
          />
        </label>

        <button type="submit" className="btn btn-primary" disabled={saving}>
          {saving ? 'Saving...' : 'Save profile'}
        </button>
      </form>
    </div>
  )
}
