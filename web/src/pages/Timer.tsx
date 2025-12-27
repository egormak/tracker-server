import { useEffect, useState, useRef } from 'react'
import Button from '@mui/material/Button'
import Grid from '@mui/material/Grid'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import MenuItem from '@mui/material/MenuItem'
import Autocomplete from '@mui/material/Autocomplete'
import AvTimerOutlinedIcon from '@mui/icons-material/AvTimerOutlined'
import PlayArrowRoundedIcon from '@mui/icons-material/PlayArrowRounded'
import PauseRoundedIcon from '@mui/icons-material/PauseRounded'
import StopRoundedIcon from '@mui/icons-material/StopRounded'
import { api, RunningTask, TaskResult } from '../api/client'
import Alert from '../components/Alert'
import Card from '../components/Card'

export default function Timer() {
  const [runningTask, setRunningTask] = useState<RunningTask | null>(null)
  const [taskName, setTaskName] = useState('')
  const [role, setRole] = useState('work')
  const [msg, setMsg] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [elapsed, setElapsed] = useState(0)
  const [availableTasks, setAvailableTasks] = useState<TaskResult[]>([])

  // Load available tasks
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

  // Timer tick
  useEffect(() => {
    let interval: number | undefined
    if (runningTask && runningTask.is_running) {
      interval = window.setInterval(() => {
        // Calculate elapsed time based on start_time
        const start = new Date(runningTask.start_time).getTime()
        const now = new Date().getTime()
        const currentSessionSeconds = Math.floor((now - start) / 1000)
        setElapsed(runningTask.accumulated * 60 + currentSessionSeconds)
      }, 1000)
    } else if (runningTask && !runningTask.is_running) {
      setElapsed(runningTask.accumulated * 60)
    } else {
      setElapsed(0)
    }
    return () => window.clearInterval(interval)
  }, [runningTask])

  const loadStatus = async () => {
    setError(null)
    try {
      const r = await api.getTaskStatus()
      if (r.data && r.data.task_name) {
        setRunningTask(r.data)
      } else {
        setRunningTask(null)
      }
    } catch (e: any) {
      // If 404 or empty, just null
      setRunningTask(null)
    }
  }

  useEffect(() => { loadStatus() }, [])

  const handleStart = async (e: React.FormEvent) => {
    e.preventDefault()
    setMsg(null); setError(null)
    if (!taskName.trim()) { setError('Task name is required'); return }
    try {
      const r = await api.startTask({ task_name: taskName, role })
      setRunningTask(r.data)
      setTaskName('')
    } catch (e: any) { setError(e.message) }
  }

  const handleStop = async () => {
    setMsg(null); setError(null)
    try {
      await api.stopTask()
      setRunningTask(null) // Clear running task
      setMsg('Task finished and saved')
      setElapsed(0)
    } catch (e: any) { setError(e.message) }
  }

  const handlePause = async () => {
    setError(null)
    try {
      const r = await api.pauseTask()
      setRunningTask(r.data)
    } catch (e: any) { setError(e.message) }
  }

  const handleResume = async () => {
    setError(null)
    try {
      const r = await api.resumeTask()
      setRunningTask(r.data)
    } catch (e: any) { setError(e.message) }
  }

  const formatTime = (totalSeconds: number) => {
    const h = Math.floor(totalSeconds / 3600)
    const m = Math.floor((totalSeconds % 3600) / 60)
    const s = totalSeconds % 60
    return `${h > 0 ? h + ':' : ''}${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`
  }

  const isRunning = runningTask !== null

  // Helper to find role for selected task
  const handleTaskChange = (newValue: string | null) => {
    setTaskName(newValue || '')
    if (newValue) {
      const found = availableTasks.find(t => t.name === newValue)
      if (found) {
        setRole(found.role)
      }
    }
  }

  return (
    <Grid container spacing={3}>
      <Grid item xs={12} md={8}>
        <Card title="Running Task" subtitle="Track your time" icon={<AvTimerOutlinedIcon />}>
          {error && <Alert type="error">{error}</Alert>}
          {msg && <Alert type="success">{msg}</Alert>}

          <Stack spacing={4} alignItems="center">
            {/* Timer Display */}
            <Typography variant="h2" sx={{ fontFamily: 'monospace', fontWeight: 'bold' }}>
              {formatTime(elapsed)}
            </Typography>

            {isRunning ? (
              <Stack spacing={2} width="100%" alignItems="center">
                <Typography variant="h5" color="text.secondary">
                  {runningTask.task_name} <Typography component="span" variant="body2" sx={{ color: 'text.disabled' }}>({runningTask.role})</Typography>
                </Typography>

                <Stack direction="row" spacing={2}>
                  {runningTask.is_running ? (
                    <Button variant="outlined" color="warning" size="large" onClick={handlePause} startIcon={<PauseRoundedIcon />}>
                      Pause
                    </Button>
                  ) : (
                    <Button variant="contained" color="success" size="large" onClick={handleResume} startIcon={<PlayArrowRoundedIcon />}>
                      Resume
                    </Button>
                  )}
                  <Button variant="contained" color="error" size="large" onClick={handleStop} startIcon={<StopRoundedIcon />}>
                    Stop
                  </Button>
                </Stack>
              </Stack>
            ) : (
              <Stack component="form" onSubmit={handleStart} spacing={2} width="100%" direction={{ xs: 'column', sm: 'row' }}>
                <Autocomplete
                  options={availableTasks.map(t => t.name)}
                  value={taskName}
                  onChange={(_, newValue) => handleTaskChange(newValue)}
                  freeSolo={false}
                  fullWidth
                  renderInput={(params) => (
                    <TextField
                      {...params}
                      label="Task Name"
                      required
                    />
                  )}
                />
                <TextField
                  select
                  label="Role"
                  value={role}
                  onChange={(e) => setRole(e.target.value)}
                  sx={{ minWidth: 120 }}
                >
                  <MenuItem value="work">Work</MenuItem>
                  <MenuItem value="learn">Learn</MenuItem>
                  <MenuItem value="rest">Rest</MenuItem>
                </TextField>
                <Button variant="contained" type="submit" size="large" startIcon={<PlayArrowRoundedIcon />}>
                  Start
                </Button>
              </Stack>
            )}

          </Stack>
        </Card>
      </Grid>
    </Grid>
  )
}
