import { useState } from 'react'
import Button from '@mui/material/Button'
import Grid from '@mui/material/Grid'
import InputAdornment from '@mui/material/InputAdornment'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import AssignmentTurnedInOutlinedIcon from '@mui/icons-material/AssignmentTurnedInOutlined'
import SaveRoundedIcon from '@mui/icons-material/SaveRounded'
import { api } from '../api/client'
import Alert from '../components/Alert'
import Card from '../components/Card'

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
    <Grid container spacing={3}>
      <Grid item xs={12} md={6}>
        <Card title="Add Task Record" subtitle="Track completed minutes" icon={<AssignmentTurnedInOutlinedIcon />}>
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
              label="Time Done (min)"
              value={timeDone}
              onChange={(e) => setTimeDone(e.target.value)}
              inputMode="numeric"
              InputProps={{ endAdornment: <InputAdornment position="end">min</InputAdornment> }}
              required
            />
            <Stack direction="row" justifyContent="flex-end">
              <Button variant="contained" type="submit" startIcon={<SaveRoundedIcon />}>
                Save
              </Button>
            </Stack>
          </Stack>
        </Card>
      </Grid>
    </Grid>
  )
}
