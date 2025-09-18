import { useEffect, useMemo, useState } from 'react'
import { api, TaskResult, RestTimeResponse } from '../api/client'
import Card from '../components/Card'
import Alert from '../components/Alert'
import Progress from '../components/Progress'

export default function Dashboard() {
  const [stats, setStats] = useState<TaskResult[]>([])
  const [rest, setRest] = useState<RestTimeResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let mounted = true
    async function load() {
      try {
        const [s, r] = await Promise.all([
          api.getStatsDoneToday(),
          api.restGet(),
        ])
        if (!mounted) return
        setStats(s)
        setRest(r)
      } catch (e: any) {
        setError(e.message)
      } finally {
        setLoading(false)
      }
    }
    load()
    return () => { mounted = false }
  }, [])

  const total = useMemo(() => {
    const done = stats.reduce((a, s) => a + (s.time_done || 0), 0)
    const plan = stats.reduce((a, s) => a + (s.time_duration || 0), 0)
    const percent = plan ? Math.round((done / plan) * 100) : 0
    return { done, plan, percent }
  }, [stats])

  return (
    <div className="grid cols-2">
      <Card title="Overview" subtitle="Today summary">
        {loading && <div className="muted">Loading…</div>}
        {error && <Alert type="error">{error}</Alert>}
        <div className="grid cols-2" style={{ marginTop: 8 }}>
          <div className="card" style={{ padding: 12 }}>
            <div className="muted">Total Done</div>
            <div className="kpi" style={{ fontSize: 24 }}>{total.done} min</div>
          </div>
          <div className="card" style={{ padding: 12 }}>
            <div className="muted">Planned</div>
            <div className="kpi" style={{ fontSize: 24 }}>{total.plan} min</div>
          </div>
        </div>
        <div style={{ marginTop: 12 }}>
          <div className="muted" style={{ marginBottom: 6 }}>Completion</div>
          <Progress value={total.percent} />
        </div>
        {rest && (
          <div className="row" style={{ marginTop: 12 }}>
            <div className="muted">Rest available</div>
            <div className="kpi">{rest.rest_time} min</div>
          </div>
        )}
      </Card>

      <Card title="Today Records" subtitle="Per task">
        {stats.length === 0 && !loading && <div className="muted">No data</div>}
        <div className="list">
          {stats.map((t) => {
            const pct = t.time_duration ? (t.time_done / t.time_duration) * 100 : 0
            return (
              <div key={t.name} className="list-item">
                <div>
                  <div style={{ fontWeight: 600 }}>{t.name}</div>
                  <div className="muted" style={{ fontSize: 12 }}>role: {t.role} · priority {t.priority}</div>
                </div>
                <div style={{ textAlign: 'right', minWidth: 160 }}>
                  <div className="kpi">{t.time_done} / {t.time_duration} min</div>
                  <Progress value={pct} />
                </div>
              </div>
            )
          })}
        </div>
      </Card>
    </div>
  )
}
