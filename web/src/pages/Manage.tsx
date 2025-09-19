import { useState } from 'react'
import Button from '@mui/material/Button'
import Grid from '@mui/material/Grid'
import MenuItem from '@mui/material/MenuItem'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import AddCircleRoundedIcon from '@mui/icons-material/AddCircleRounded'
import ManageAccountsRoundedIcon from '@mui/icons-material/ManageAccountsRounded'
import { api } from '../api/client'
import Alert from '../components/Alert'
import Card from '../components/Card'

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
    <Grid container spacing={3}>
      <Grid item xs={12} md={6}>
        <Card title="Create Task" subtitle="Add new task with role" icon={<ManageAccountsRoundedIcon />}>
          {error && <Alert type="error">{error}</Alert>}
          {msg && <Alert type="success">{msg}</Alert>}
          <Stack component="form" onSubmit={submit} spacing={2} sx={{ maxWidth: 460 }}>
            <TextField
              label="Task Name"
              value={taskName}
              onChange={(e) => setTaskName(e.target.value)}
              required
            />
            <TextField
              select
              label="Role"
              value={role}
              onChange={(e) => setRole(e.target.value)}
            >
              <MenuItem value="work">work</MenuItem>
              <MenuItem value="learn">learn</MenuItem>
              <MenuItem value="rest">rest</MenuItem>
              <MenuItem value="plan">plan</MenuItem>
            </TextField>
            <Stack direction="row" justifyContent="flex-end">
              <Button variant="contained" type="submit" startIcon={<AddCircleRoundedIcon />}>
                Create Task
              </Button>
            </Stack>
          </Stack>
        </Card>
      </Grid>
    </Grid>
  )
}
