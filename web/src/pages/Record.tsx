import { useState, useEffect } from 'react'
import Button from '@mui/material/Button'
import FormControl from '@mui/material/FormControl'
import Grid from '@mui/material/Grid'
import InputAdornment from '@mui/material/InputAdornment'
import InputLabel from '@mui/material/InputLabel'
import MenuItem from '@mui/material/MenuItem'
import Select from '@mui/material/Select'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import Autocomplete from '@mui/material/Autocomplete'
import AssignmentTurnedInOutlinedIcon from '@mui/icons-material/AssignmentTurnedInOutlined'
import SaveRoundedIcon from '@mui/icons-material/SaveRounded'
import { api, TaskResult } from '../api/client'
import Alert from '../components/Alert'
import Card from '../components/Card'

const DAYS_OF_WEEK = ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday']

export default function Record() {
  const [taskName, setTaskName] = useState('')
  const [timeDone, setTimeDone] = useState('')
  const [sourceDay, setSourceDay] = useState('')
  const [msg, setMsg] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [availableTasks, setAvailableTasks] = useState<TaskResult[]>([])

  useEffect(() => {
    const loadTasks = async () => {
      try {
        const tasks = await api.getTaskList()
        setAvailableTasks(tasks)
      } catch (e) {
        console.error("Failed to load tasks", e)
      }
    }
    loadTasks()
  }, [])

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setMsg(null); setError(null)
    const minutes = parseInt(timeDone, 10)
    if (!taskName) { setError('Task name required'); return }
    if (Number.isNaN(minutes) || minutes <= 0) { setError('Time must be positive integer'); return }
    try {
      const payload: { task_name: string; time_done: number; source_day?: string } = {
        task_name: taskName,
        time_done: minutes
      }
      if (sourceDay) {
        payload.source_day = sourceDay
      }
      await api.addTaskRecord(payload)
      setMsg('Record saved')
      setTaskName('')
      setTimeDone('')
      setSourceDay('')
    } catch (e: any) { setError(e.message) }
  }

  // Find selected task role if possible, though currently not needed for submit, 
  // backend handles role lookup if not provided, but addTaskRecord API usually takes just name.

  return (
    <Grid container spacing={3}>
      <Grid item xs={12} md={6}>
        <Card title="Add Task Record" subtitle="Track completed minutes" icon={<AssignmentTurnedInOutlinedIcon />}>
          {error && <Alert type="error">{error}</Alert>}
          {msg && <Alert type="success">{msg}</Alert>}
          <Stack component="form" onSubmit={submit} spacing={2} sx={{ maxWidth: 460 }}>

            <Autocomplete
              options={availableTasks.map(t => t.name)}
              value={taskName}
              onChange={(_, newValue) => setTaskName(newValue || '')}
              freeSolo={false} // Restrict to existing tasks
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Task Name"
                  required
                />
              )}
            />

            <TextField
              label="Time Done (min)"
              value={timeDone}
              onChange={(e) => setTimeDone(e.target.value)}
              inputMode="numeric"
              InputProps={{ endAdornment: <InputAdornment position="end">min</InputAdornment> }}
              required
            />
            <FormControl>
              <InputLabel id="source-day-label">Source Day (optional)</InputLabel>
              <Select
                labelId="source-day-label"
                label="Source Day (optional)"
                value={sourceDay}
                onChange={(e) => setSourceDay(e.target.value)}
              >
                <MenuItem value="">
                  <em>Today</em>
                </MenuItem>
                {DAYS_OF_WEEK.map((day) => (
                  <MenuItem key={day} value={day}>
                    {day.charAt(0).toUpperCase() + day.slice(1)}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
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
