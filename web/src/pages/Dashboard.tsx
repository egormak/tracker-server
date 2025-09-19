import { useEffect, useMemo, useState } from 'react'
import { api, RestTimeResponse, RecordsSummary, TaskResult } from '../api/client'
import Card from '../components/Card'
import Alert from '../components/Alert'
import Progress from '../components/Progress'

export default function Dashboard() {
  const [records, setRecords] = useState<RecordsSummary | null>(null)
  const [rest, setRest] = useState<RestTimeResponse | null>(null)
  const [tasks, setTasks] = useState<TaskResult[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let mounted = true
    async function load() {
      try {
        const [rec, r, tl] = await Promise.all([
          api.getRecordsSummary(),
          api.restGet(),
          api.getStatsTasksToday(),
        ])
        if (!mounted) return
        setRecords(rec)
        setRest(r)
        setTasks(tl)
      } catch (e: any) {
        setError(e.message)
      } finally {
        setLoading(false)
      }
    }
    load()
    return () => { mounted = false }
  }, [])

  const totals = useMemo(() => {
    const sum = (m?: Record<string, number>) => Object.values(m || {}).reduce((a, b) => a + b, 0)
    const today = sum(records?.today)
    const yesterday = sum(records?.yesterday)
    const all = sum(records?.all)
    const vsYesterday = yesterday ? Math.round((today / yesterday) * 100) : 0
    return { today, yesterday, all, vsYesterday }
  }, [records])

  return (
    <div className="grid cols-2">
      <Card title="Overview" subtitle="Today summary">
        {loading && <div className="muted">Loading…</div>}
        {error && <Alert type="error">{error}</Alert>}
        <div className="grid cols-2" style={{ marginTop: 8 }}>
          <div className="card" style={{ padding: 12 }}>
            <div className="muted">Total Today</div>
            <div className="kpi" style={{ fontSize: 24 }}>{totals.today} min</div>
          </div>
          <div className="card" style={{ padding: 12 }}>
            <div className="muted">Yesterday</div>
            <div className="kpi" style={{ fontSize: 24 }}>{totals.yesterday} min</div>
          </div>
        </div>
        <div style={{ marginTop: 12 }}>
          <div className="muted" style={{ marginBottom: 6 }}>Today vs Yesterday</div>
          <Progress value={totals.vsYesterday} />
        </div>
        <div className="muted" style={{ marginTop: 6 }}>All time: {totals.all} min</div>
        {rest && (
          <div className="row" style={{ marginTop: 12 }}>
            <div className="muted">Rest available</div>
            <div className="kpi">{rest.rest_time} min</div>
          </div>
        )}
      </Card>

      <Card title="Today Records" subtitle="Per category (aggregated)">
        {(!records || Object.keys(records.today || {}).length === 0) && !loading && <div className="muted">No data</div>}
        <div className="list">
          {records && Object.entries(records.today).sort((a, b) => b[1] - a[1]).map(([name, minutes]) => (
            <div key={name} className="list-item">
              <div>
                <div style={{ fontWeight: 600 }}>{name}</div>
              </div>
              <div style={{ textAlign: 'right', minWidth: 160 }}>
                <div className="kpi">{minutes} min</div>
              </div>
            </div>
          ))}
        </div>
      </Card>

      <Card title="Tasks by Plan" subtitle="Planned vs done (today)">
        {tasks.length === 0 && !loading && <div className="muted">No tasks</div>}
        <div className="list">
          {tasks.map((t) => {
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
