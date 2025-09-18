import { useState } from 'react'
import { api } from '../api/client'
import Card from '../components/Card'
import Alert from '../components/Alert'

export default function Record() {
  const [taskName, setTaskName] = useState('')
  const [timeDone, setTimeDone] = useState('')
  const [msg, setMsg] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setMsg(null); setError(null)
    const minutes = parseInt(timeDone, 10)
    if (!taskName) { setError('Task name required'); return }
    if (Number.isNaN(minutes) || minutes <= 0) { setError('Time must be positive integer'); return }
    try {
      await api.addTaskRecord({ task_name: taskName, time_done: minutes })
      setMsg('Record saved')
      setTaskName('')
      setTimeDone('')
    } catch (e: any) { setError(e.message) }
  }

  return (
    <div className="grid cols-2">
      <Card title="Add Task Record" subtitle="Track completed minutes">
        {error && <Alert type="error">{error}</Alert>}
        {msg && <Alert type="success">{msg}</Alert>}
        <form onSubmit={submit} className="grid" style={{ gap: 12, maxWidth: 460 }}>
          <div className="field">
            <label className="label">Task Name</label>
            <input className="input" value={taskName} onChange={e => setTaskName(e.target.value)} />
          </div>
          <div className="field">
            <label className="label">Time Done (min)</label>
            <input className="input" value={timeDone} onChange={e => setTimeDone(e.target.value)} />
          </div>
          <div className="row" style={{ justifyContent: 'flex-end' }}>
            <button className="btn primary" type="submit">Save</button>
          </div>
        </form>
      </Card>
    </div>
  )
}
