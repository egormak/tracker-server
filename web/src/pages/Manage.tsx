import { useState } from 'react'
import { api } from '../api/client'
import Card from '../components/Card'
import Alert from '../components/Alert'

export default function Manage() {
  const [taskName, setTaskName] = useState('')
  const [role, setRole] = useState('work')
  const [msg, setMsg] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setMsg(null); setError(null)
    if (!taskName) { setError('Task name required'); return }
    try {
      await api.createTask({ task_name: taskName, role })
      setMsg('Task created')
      setTaskName('')
    } catch (e: any) { setError(e.message) }
  }

  return (
    <div className="grid cols-2">
      <Card title="Create Task" subtitle="Add new task with role">
        {error && <Alert type="error">{error}</Alert>}
        {msg && <Alert type="success">{msg}</Alert>}
        <form onSubmit={submit} className="grid" style={{ gap: 12, maxWidth: 460 }}>
          <div className="field">
            <label className="label">Task Name</label>
            <input className="input" value={taskName} onChange={e => setTaskName(e.target.value)} />
          </div>
          <div className="field">
            <label className="label">Role</label>
            <select className="select" value={role} onChange={e => setRole(e.target.value)}>
              <option value="work">work</option>
              <option value="learn">learn</option>
              <option value="rest">rest</option>
              <option value="plan">plan</option>
            </select>
          </div>
          <div className="row" style={{ justifyContent: 'flex-end' }}>
            <button className="btn primary" type="submit">Create Task</button>
          </div>
        </form>
      </Card>
    </div>
  )
}
