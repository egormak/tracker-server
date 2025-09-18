import { PropsWithChildren } from 'react'

export default function Card({ title, subtitle, children }: PropsWithChildren<{ title?: string; subtitle?: string }>) {
  return (
    <div className="card">
      {title && <h3>{title}</h3>}
      {subtitle && <div className="muted" style={{ marginBottom: 8 }}>{subtitle}</div>}
      {children}
    </div>
  )
}

