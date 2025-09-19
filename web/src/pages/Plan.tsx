import { useEffect, useState } from 'react'
import Button from '@mui/material/Button'
import Chip from '@mui/material/Chip'
import Grid from '@mui/material/Grid'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import RefreshRoundedIcon from '@mui/icons-material/RefreshRounded'
import PlaylistAddCircleRoundedIcon from '@mui/icons-material/PlaylistAddCircleRounded'
import TuneOutlinedIcon from '@mui/icons-material/TuneOutlined'
import SaveRoundedIcon from '@mui/icons-material/SaveRounded'
import { api, PlanPercentResponse } from '../api/client'
import Alert from '../components/Alert'
import Card from '../components/Card'

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
    <Grid container spacing={3} alignItems="stretch">
      <Grid item xs={12} md={6}>
        <Card
          title="Next by Plan"
          subtitle="Based on plan percent"
          icon={<PlaylistAddCircleRoundedIcon />}
          actions={
            plan && (
              <Button variant="outlined" onClick={rotate} startIcon={<RefreshRoundedIcon />}>
                Rotate plan group (legacy)
              </Button>
            )
          }
        >
          {error && <Alert type="error">{error}</Alert>}
          {msg && <Alert type="success">{msg}</Alert>}
          {plan ? (
            <Stack spacing={3}>
              <Stack spacing={1}>
                <Typography variant="h5" fontWeight={600}>
                  {plan.task_name}
                </Typography>
                <Stack direction="row" spacing={1} alignItems="center">
                  <Chip label={`Time left: ${plan.time_left} min`} size="small" color="secondary" variant="outlined" />
                  <Chip label={`${plan.percent}% ready`} size="small" sx={{ bgcolor: 'rgba(99,102,241,0.16)' }} />
                </Stack>
              </Stack>
              <Typography variant="h2" sx={{ fontWeight: 700 }}>
                {plan.percent}%
              </Typography>
              <Typography variant="body2" color="text.secondary">
                Legacy rotation keeps tasks balanced across plan groups.
              </Typography>
            </Stack>
          ) : (
            <Typography color="text.secondary">No plan data.</Typography>
          )}
        </Card>
      </Grid>

      <Grid item xs={12} md={6}>
        <Card title="Set Procents" subtitle="Comma-separated values; optional role" icon={<TuneOutlinedIcon />}>
          {error && <Alert type="error">{error}</Alert>}
          {msg && <Alert type="success">{msg}</Alert>}
          <Stack component="form" onSubmit={submitProcents} spacing={2}>
            <TextField
              label="Procents"
              value={procents}
              onChange={(e) => setProcents(e.target.value)}
              placeholder="80,20,0"
            />
            <TextField
              label="Role (plan | work | learn | rest)"
              value={role}
              onChange={(e) => setRole(e.target.value)}
              placeholder="plan"
            />
            <Stack direction="row" justifyContent="flex-end" spacing={1}>
              <Button variant="contained" type="submit" startIcon={<SaveRoundedIcon />}>
                Save Procents
              </Button>
            </Stack>
          </Stack>
        </Card>
      </Grid>
    </Grid>
  )
}
