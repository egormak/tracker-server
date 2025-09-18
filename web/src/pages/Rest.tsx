import { useEffect, useState } from 'react'
import { api } from '../api/client'
import Card from '../components/Card'
import Alert from '../components/Alert'

export default function Rest() {
  const [rest, setRest] = useState<number | null>(null)
  const [value, setValue] = useState('')
  const [msg, setMsg] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const load = async () => {
    setError(null)
    try {
      const r = await api.restGet()
      setRest(r.rest_time)
    } catch (e: any) { setError(e.message) }
  }
  useEffect(() => { load() }, [])

  const act = async (action: 'add' | 'spend') => {
    setMsg(null); setError(null)
    const n = parseInt(value, 10)
    if (Number.isNaN(n) || n <= 0) { setError('Enter positive integer minutes'); return }
    try {
      if (action === 'add') await api.restAdd({ rest_time: n })
      else await api.restSpend({ rest_time: n })
      await load()
      setMsg(`${action === 'add' ? 'Added' : 'Spent'} ${n} minutes`)
      setValue('')
    } catch (e: any) { setError(e.message) }
  }

  return (
    <div className="grid cols-2">
      <Card title="Rest Balance" subtitle="Manage rest minutes">
        {error && <Alert type="error">{error}</Alert>}
        {msg && <Alert type="success">{msg}</Alert>}
        <div className="kpi" style={{ fontSize: 28, marginBottom: 8 }}>{rest ?? '-'} min</div>
        <div className="row">
          <input className="input" value={value} onChange={e => setValue(e.target.value)} placeholder="minutes" style={{ maxWidth: 160 }} />
          <button className="btn primary" onClick={() => act('add')}>Add</button>
          <button className="btn" onClick={() => act('spend')}>Spend</button>
        </div>
      </Card>
    </div>
  )
}
