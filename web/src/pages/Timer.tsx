import { useEffect, useState } from 'react'
import Button from '@mui/material/Button'
import Grid from '@mui/material/Grid'
import InputAdornment from '@mui/material/InputAdornment'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import AvTimerOutlinedIcon from '@mui/icons-material/AvTimerOutlined'
import PlayArrowRoundedIcon from '@mui/icons-material/PlayArrowRounded'
import { api } from '../api/client'
import Alert from '../components/Alert'
import Card from '../components/Card'

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
    <Grid container spacing={3}>
      <Grid item xs={12} md={6}>
        <Card title="Timer" subtitle="Current duration and settings" icon={<AvTimerOutlinedIcon />}>
          {error && <Alert type="error">{error}</Alert>}
          {msg && <Alert type="success">{msg}</Alert>}
          <Stack spacing={3}>
            <Typography variant="h3">{current ?? '-'} min</Typography>
            <Stack component="form" onSubmit={submit} direction={{ xs: 'column', sm: 'row' }} spacing={2} alignItems="flex-start">
              <TextField
                label="Count"
                value={count}
                onChange={(e) => setCount(e.target.value)}
                inputMode="numeric"
                InputProps={{ endAdornment: <InputAdornment position="end">cycles</InputAdornment> }}
                sx={{ maxWidth: 200 }}
              />
              <Button variant="contained" type="submit" startIcon={<PlayArrowRoundedIcon />}>
                Set
              </Button>
            </Stack>
          </Stack>
        </Card>
      </Grid>
    </Grid>
  )
}
