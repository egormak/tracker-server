import { NavLink } from 'react-router-dom'

export default function Header() {
  return (
    <div className="header">
      <div className="header-inner container">
        <div className="brand">
          <div className="brand-badge">T</div>
          <div>Tracker</div>
        </div>
        <nav className="nav">
          <NavLink to="/" end className={({ isActive }) => isActive ? 'active' : ''}>Dashboard</NavLink>
          <NavLink to="/plan" className={({ isActive }) => isActive ? 'active' : ''}>Plan</NavLink>
          <NavLink to="/rest" className={({ isActive }) => isActive ? 'active' : ''}>Rest</NavLink>
          <NavLink to="/record" className={({ isActive }) => isActive ? 'active' : ''}>Record</NavLink>
          <NavLink to="/manage" className={({ isActive }) => isActive ? 'active' : ''}>Manage</NavLink>
          <NavLink to="/timer" className={({ isActive }) => isActive ? 'active' : ''}>Timer</NavLink>
        </nav>
      </div>
    </div>
  )
}

