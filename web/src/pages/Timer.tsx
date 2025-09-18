import { useEffect, useState } from 'react'
import { api } from '../api/client'
import Card from '../components/Card'
import Alert from '../components/Alert'

export default function Timer() {
  const [current, setCurrent] = useState<number | null>(null)
  const [count, setCount] = useState('')
  const [msg, setMsg] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const load = async () => {
    setError(null)
    try {
      const r = await api.timerGet()
      setCurrent(r.time_duration)
    } catch (e: any) { setError(e.message) }
  }
  useEffect(() => { load() }, [])

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setMsg(null); setError(null)
    const n = parseInt(count, 10)
    if (Number.isNaN(n) || n <= 0) { setError('Count must be positive integer'); return }
    try {
      await api.timerSet({ count: n })
      setMsg('Timer set')
      setCount('')
      await load()
    } catch (e: any) { setError(e.message) }
  }

  return (
    <div className="grid cols-2">
      <Card title="Timer" subtitle="Current duration and settings">
        {error && <Alert type="error">{error}</Alert>}
        {msg && <Alert type="success">{msg}</Alert>}
        <div className="kpi" style={{ fontSize: 28, marginBottom: 8 }}>{current ?? '-'} min</div>
        <form onSubmit={submit} className="row">
          <input className="input" value={count} onChange={e => setCount(e.target.value)} placeholder="count" style={{ maxWidth: 160 }} />
          <button className="btn primary" type="submit">Set</button>
        </form>
      </Card>
    </div>
  )
}
