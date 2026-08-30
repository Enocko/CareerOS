import { Link, useLocation } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { NotificationBell } from './NotificationBell'

export function Navbar() {
  const { user, logout, isAuthenticated } = useAuth()
  const location = useLocation()

  const linkClass = (path: string) =>
    `nav-link${location.pathname.startsWith(path) ? ' active' : ''}`

  return (
    <header className="navbar">
      <div className="container navbar-inner">
        <Link to="/" className="brand">
          CareerOS
        </Link>

        {isAuthenticated && (
          <nav className="nav-links">
            <Link to="/recommended" className={linkClass('/recommended')}>
              For You
            </Link>
            <Link to="/browse" className={linkClass('/browse')}>
              Browse
            </Link>
            <Link to="/saved" className={linkClass('/saved')}>
              Saved
            </Link>
            <Link to="/applications" className={linkClass('/applications')}>
              Applications
            </Link>
            <Link to="/profile" className={linkClass('/profile')}>
              Profile
            </Link>
          </nav>
        )}

        <div className="nav-actions">
          {isAuthenticated && <NotificationBell />}
          {isAuthenticated ? (
            <>
              <span className="user-email">{user?.email}</span>
              <button type="button" className="btn btn-secondary" onClick={logout}>
                Log out
              </button>
            </>
          ) : (
            <>
              <Link to="/login" className="btn btn-secondary">
                Log in
              </Link>
              <Link to="/register" className="btn btn-primary">
                Register
              </Link>
            </>
          )}
        </div>
      </div>
    </header>
  )
}
