import { useEffect, useState } from 'react'
import {
  Box,
  Button,
  Card as MuiCard,
  CardContent,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Grid,
  IconButton,
  Stack,
  Tab,
  Tabs,
  TextField,
  Typography,
  MenuItem,
  Divider,
} from '@mui/material'
import AddIcon from '@mui/icons-material/Add'
import DeleteIcon from '@mui/icons-material/Delete'
import EditIcon from '@mui/icons-material/Edit'
import SaveIcon from '@mui/icons-material/Save'
import CheckCircleIcon from '@mui/icons-material/CheckCircle'
import CalendarTodayIcon from '@mui/icons-material/CalendarToday'
import { api, DaySchedule, ScheduleTask, WeeklySchedule, ActiveSchedule } from '../api/client'
import Alert from '../components/Alert'

const DAYS = ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday']
const DAY_LABELS: Record<string, string> = {
  monday: 'Monday',
  tuesday: 'Tuesday',
  wednesday: 'Wednesday',
  thursday: 'Thursday',
  friday: 'Friday',
  saturday: 'Saturday',
  sunday: 'Sunday',
}

const ROLES = ['work', 'learn', 'rest'] as const

const emptyTask = (): ScheduleTask => ({
  name: '',
  role: 'work',
  time: 0,
  priority: 5,
  percents: [],
})

const emptyDaySchedule = (day: string): DaySchedule => ({
  day,
  total_time: 900,
  tasks: [],
  plan_group: ['plan', 'work', 'learn', 'rest'],
})

const emptyWeekSchedule = (): Omit<WeeklySchedule, 'id' | 'title' | 'created_at' | 'updated_at' | 'is_active'> => ({
  monday: emptyDaySchedule('monday'),
  tuesday: emptyDaySchedule('tuesday'),
  wednesday: emptyDaySchedule('wednesday'),
  thursday: emptyDaySchedule('thursday'),
  friday: emptyDaySchedule('friday'),
  saturday: emptyDaySchedule('saturday'),
  sunday: emptyDaySchedule('sunday'),
})

export default function Schedule() {
  const [activeTab, setActiveTab] = useState(0)
  const [schedule, setSchedule] = useState<WeeklySchedule | null>(null)
  const [editSchedule, setEditSchedule] = useState(emptyWeekSchedule())
  const [isEditing, setIsEditing] = useState(false)
  const [todaySchedule, setTodaySchedule] = useState<ActiveSchedule | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [taskDialog, setTaskDialog] = useState<{
    open: boolean
    dayIndex: number
    taskIndex: number | null
    task: ScheduleTask
  }>({ open: false, dayIndex: 0, taskIndex: null, task: emptyTask() })

  const currentDay = DAYS[activeTab]

  useEffect(() => {
    loadSchedule()
    loadTodaySchedule()
  }, [])

  async function loadSchedule() {
    try {
      const response = await api.getActiveSchedule()
      setSchedule(response.data)
      setEditSchedule(response.data)
      setLoading(false)
    } catch (e: any) {
      if (e.message.includes('no active schedule')) {
        setSchedule(null)
        setEditSchedule(emptyWeekSchedule())
        setIsEditing(true)
      } else {
        setError(e.message)
      }
      setLoading(false)
    }
  }

  async function loadTodaySchedule() {
    try {
      const response = await api.getTodaySchedule()
      setTodaySchedule(response.data)
    } catch (e: any) {
      console.warn('Could not load today schedule:', e.message)
    }
  }

  async function handleSave() {
    try {
      setError(null)
      if (schedule?.id) {
        await api.updateSchedule(schedule.id, editSchedule)
        setSuccess('Schedule updated successfully')
      } else {
        const response = await api.createSchedule({ schedule: editSchedule, set_active: true })
        setSuccess(`Schedule created and activated: ${response.data.schedule_id}`)
      }
      setIsEditing(false)
      await loadSchedule()
      await loadTodaySchedule()
    } catch (e: any) {
      setError(e.message)
    }
  }

  async function handleApply() {
    try {
      setError(null)
      await api.applySchedule()
      setSuccess('Schedule applied for today! Tasks created.')
      await loadTodaySchedule()
    } catch (e: any) {
      setError(e.message)
    }
  }

  function handleAddTask() {
    setTaskDialog({
      open: true,
      dayIndex: activeTab,
      taskIndex: null,
      task: emptyTask(),
    })
  }

  function handleEditTask(taskIndex: number) {
    const day = DAYS[activeTab] as keyof typeof editSchedule
    setTaskDialog({
      open: true,
      dayIndex: activeTab,
      taskIndex,
      task: { ...editSchedule[day].tasks[taskIndex] },
    })
  }

  function handleDeleteTask(taskIndex: number) {
    const day = DAYS[activeTab] as keyof typeof editSchedule
    setEditSchedule({
      ...editSchedule,
      [day]: {
        ...editSchedule[day],
        tasks: editSchedule[day].tasks.filter((_, i) => i !== taskIndex),
      },
    })
  }

  function handleSaveTask() {
    const day = DAYS[taskDialog.dayIndex] as keyof typeof editSchedule
    const daySchedule = editSchedule[day]
    
    if (taskDialog.taskIndex !== null) {
      // Edit existing task
      const newTasks = [...daySchedule.tasks]
      newTasks[taskDialog.taskIndex] = taskDialog.task
      setEditSchedule({
        ...editSchedule,
        [day]: { ...daySchedule, tasks: newTasks },
      })
    } else {
      // Add new task
      setEditSchedule({
        ...editSchedule,
        [day]: { ...daySchedule, tasks: [...daySchedule.tasks, taskDialog.task] },
      })
    }
    
    setTaskDialog({ ...taskDialog, open: false })
  }

  function updateDayTotalTime(time: number) {
    const day = DAYS[activeTab] as keyof typeof editSchedule
    setEditSchedule({
      ...editSchedule,
      [day]: { ...editSchedule[day], total_time: time },
    })
  }

  const currentDaySchedule = editSchedule[currentDay as keyof typeof editSchedule]
  const totalScheduledTime = currentDaySchedule.tasks.reduce((sum, t) => sum + t.time, 0)

  if (loading) {
    return (
      <Box sx={{ p: 3 }}>
        <Typography>Loading schedule...</Typography>
      </Box>
    )
  }

  return (
    <Box sx={{ p: 3 }}>
      <Stack spacing={3}>
        {/* Header */}
        <Stack direction="row" justifyContent="space-between" alignItems="center">
          <Stack direction="row" spacing={2} alignItems="center">
            <CalendarTodayIcon sx={{ fontSize: 32 }} />
            <Typography variant="h4">Weekly Schedule</Typography>
            {schedule?.is_active && (
              <Chip
                icon={<CheckCircleIcon />}
                label="Active"
                color="success"
                size="small"
              />
            )}
          </Stack>
          <Stack direction="row" spacing={2}>
            {!isEditing && schedule && (
              <Button
                variant="outlined"
                startIcon={<EditIcon />}
                onClick={() => setIsEditing(true)}
              >
                Edit
              </Button>
            )}
            {isEditing && (
              <Button
                variant="contained"
                startIcon={<SaveIcon />}
                onClick={handleSave}
              >
                Save Schedule
              </Button>
            )}
            {schedule && (
              <Button variant="contained" color="secondary" onClick={handleApply}>
                Apply Today
              </Button>
            )}
          </Stack>
        </Stack>

        {error && <Alert type="error">{error}</Alert>}
        {success && <Alert type="success">{success}</Alert>}

        {/* Today's Schedule with Rollovers */}
        {todaySchedule && todaySchedule.rollover_tasks && todaySchedule.rollover_tasks.length > 0 && (
          <MuiCard>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Rollover Tasks (Incomplete from Previous Days)
              </Typography>
              <Stack spacing={1}>
                {todaySchedule.rollover_tasks.map((task, i) => (
                  <Box
                    key={i}
                    sx={{
                      p: 2,
                      bgcolor: 'warning.light',
                      borderRadius: 1,
                      display: 'flex',
                      justifyContent: 'space-between',
                    }}
                  >
                    <Box>
                      <Typography variant="body1" fontWeight="bold">
                        {task.task_name}
                      </Typography>
                      <Typography variant="body2" color="text.secondary">
                        From {task.source_day} • Priority: {task.priority} • Role: {task.role}
                      </Typography>
                    </Box>
                    <Chip
                      label={`${task.remaining_time} min left`}
                      size="small"
                      color="warning"
                    />
                  </Box>
                ))}
              </Stack>
            </CardContent>
          </MuiCard>
        )}

        {/* Day Tabs */}
        <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)} variant="scrollable">
          {DAYS.map((day, i) => (
            <Tab key={day} label={DAY_LABELS[day]} />
          ))}
        </Tabs>

        {/* Day Configuration */}
        <MuiCard>
          <CardContent>
            <Stack spacing={3}>
              {/* Day Header */}
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Typography variant="h6">{DAY_LABELS[currentDay]}</Typography>
                {isEditing && (
                  <TextField
                    label="Total Time (minutes)"
                    type="number"
                    value={currentDaySchedule.total_time}
                    onChange={(e) => updateDayTotalTime(Number(e.target.value))}
                    size="small"
                    sx={{ width: 200 }}
                  />
                )}
              </Stack>

              <Stack direction="row" spacing={2} alignItems="center">
                <Typography variant="body2" color="text.secondary">
                  Total Scheduled: {totalScheduledTime} min
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  Total Available: {currentDaySchedule.total_time} min
                </Typography>
                {totalScheduledTime > currentDaySchedule.total_time && (
                  <Chip
                    label="Over budget!"
                    color="error"
                    size="small"
                  />
                )}
              </Stack>

              <Divider />

              {/* Tasks */}
              <Stack spacing={2}>
                {currentDaySchedule.tasks.length === 0 ? (
                  <Typography color="text.secondary">No tasks scheduled for this day</Typography>
                ) : (
                  currentDaySchedule.tasks.map((task, i) => (
                    <Box
                      key={i}
                      sx={{
                        p: 2,
                        border: 1,
                        borderColor: 'divider',
                        borderRadius: 1,
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                      }}
                    >
                      <Stack spacing={1} flex={1}>
                        <Stack direction="row" spacing={2} alignItems="center">
                          <Typography variant="body1" fontWeight="bold">
                            {task.name}
                          </Typography>
                          <Chip label={task.role} size="small" />
                          <Chip label={`Priority: ${task.priority}`} size="small" variant="outlined" />
                        </Stack>
                        <Typography variant="body2" color="text.secondary">
                          Time: {task.time} minutes
                          {task.percents && task.percents.length > 0 && (
                            <> • Percents: {task.percents.join(', ')}</>
                          )}
                        </Typography>
                      </Stack>
                      {isEditing && (
                        <Stack direction="row" spacing={1}>
                          <IconButton size="small" onClick={() => handleEditTask(i)}>
                            <EditIcon />
                          </IconButton>
                          <IconButton size="small" color="error" onClick={() => handleDeleteTask(i)}>
                            <DeleteIcon />
                          </IconButton>
                        </Stack>
                      )}
                    </Box>
                  ))
                )}
              </Stack>

              {isEditing && (
                <Button
                  variant="outlined"
                  startIcon={<AddIcon />}
                  onClick={handleAddTask}
                >
                  Add Task
                </Button>
              )}
            </Stack>
          </CardContent>
        </MuiCard>
      </Stack>

      {/* Task Dialog */}
      <Dialog open={taskDialog.open} onClose={() => setTaskDialog({ ...taskDialog, open: false })} maxWidth="sm" fullWidth>
        <DialogTitle>{taskDialog.taskIndex !== null ? 'Edit Task' : 'Add Task'}</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <TextField
              label="Task Name"
              value={taskDialog.task.name}
              onChange={(e) => setTaskDialog({ ...taskDialog, task: { ...taskDialog.task, name: e.target.value } })}
              fullWidth
            />
            <TextField
              select
              label="Role"
              value={taskDialog.task.role}
              onChange={(e) => setTaskDialog({ ...taskDialog, task: { ...taskDialog.task, role: e.target.value as any } })}
              fullWidth
            >
              {ROLES.map((role) => (
                <MenuItem key={role} value={role}>
                  {role}
                </MenuItem>
              ))}
            </TextField>
            <TextField
              label="Time (minutes)"
              type="number"
              value={taskDialog.task.time}
              onChange={(e) => setTaskDialog({ ...taskDialog, task: { ...taskDialog.task, time: Number(e.target.value) } })}
              fullWidth
            />
            <TextField
              label="Priority (1-10)"
              type="number"
              value={taskDialog.task.priority}
              onChange={(e) => setTaskDialog({ ...taskDialog, task: { ...taskDialog.task, priority: Number(e.target.value) } })}
              fullWidth
              helperText="Higher number = higher priority"
            />
            <TextField
              label="Percents (comma-separated)"
              value={taskDialog.task.percents?.join(',') || ''}
              onChange={(e) => {
                const percents = e.target.value
                  .split(',')
                  .map((p) => parseInt(p.trim()))
                  .filter((p) => !isNaN(p))
                setTaskDialog({ ...taskDialog, task: { ...taskDialog.task, percents } })
              }}
              fullWidth
              helperText="e.g., 10,20,30,40,50,60,70,80,90,100"
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setTaskDialog({ ...taskDialog, open: false })}>Cancel</Button>
          <Button onClick={handleSaveTask} variant="contained">
            Save
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}
