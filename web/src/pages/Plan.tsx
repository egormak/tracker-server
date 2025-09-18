import { useEffect, useState } from 'react'
import { api, PlanPercentResponse } from '../api/client'
import Card from '../components/Card'
import Alert from '../components/Alert'

export default function Plan() {
  const [plan, setPlan] = useState<PlanPercentResponse | null>(null)
  const [procents, setProcents] = useState('')
  const [role, setRole] = useState('')
  const [msg, setMsg] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const load = async () => {
    setError(null)
    try {
      const p = await api.getTaskPlanPercent()
      setPlan(p)
    } catch (e: any) {
      setError(e.message)
    }
  }

  useEffect(() => { load() }, [])

  const rotate = async () => {
    setMsg(null); setError(null)
    try {
      await api.changeLegacyPlanPercent()
      await load()
      setMsg('Rotated plan percent group')
    } catch (e: any) { setError(e.message) }
  }

  const submitProcents = async (e: React.FormEvent) => {
    e.preventDefault()
    setMsg(null); setError(null)
    try {
      const arr = procents.split(',').map(s => parseInt(s.trim(), 10)).filter(n => !Number.isNaN(n))
      if (!arr.length) throw new Error('Provide comma-separated integers')
      await api.setProcents(arr, role || undefined)
      setMsg('Procents saved')
    } catch (e: any) { setError(e.message) }
  }

  return (
    <div className="grid cols-2">
      <Card title="Next by Plan" subtitle="Based on plan percent">
        {error && <Alert type="error">{error}</Alert>}
        {msg && <Alert type="success">{msg}</Alert>}
        {plan ? (
          <div className="grid">
            <div className="list-item">
              <div>
                <div style={{ fontWeight: 600 }}>{plan.task_name}</div>
                <div className="muted" style={{ fontSize: 12 }}>time left: {plan.time_left} min</div>
              </div>
              <div className="kpi">{plan.percent}%</div>
            </div>
            <div className="row" style={{ justifyContent: 'flex-end' }}>
              <button className="btn" onClick={rotate}>Rotate plan group (legacy)</button>
            </div>
          </div>
        ) : (
          <div className="muted">No plan data.</div>
        )}
      </Card>

      <Card title="Set Procents" subtitle="Comma-separated values; optional role">
        {error && <Alert type="error">{error}</Alert>}
        {msg && <Alert type="success">{msg}</Alert>}
        <form onSubmit={submitProcents} className="grid" style={{ gap: 12 }}>
          <div className="field">
            <label className="label">Procents</label>
            <input className="input" value={procents} onChange={e => setProcents(e.target.value)} placeholder="80,20,0" />
          </div>
          <div className="field">
            <label className="label">Role (plan | work | learn | rest)</label>
            <input className="input" value={role} onChange={e => setRole(e.target.value)} placeholder="plan" />
          </div>
          <div className="row" style={{ justifyContent: 'flex-end' }}>
            <button className="btn primary" type="submit">Save Procents</button>
          </div>
        </form>
      </Card>
    </div>
  )
}
