import { useEffect, useState } from 'react'
import Button from '@mui/material/Button'
import Grid from '@mui/material/Grid'
import InputAdornment from '@mui/material/InputAdornment'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import LocalHotelOutlinedIcon from '@mui/icons-material/LocalHotelOutlined'
import AddRoundedIcon from '@mui/icons-material/AddRounded'
import RemoveRoundedIcon from '@mui/icons-material/RemoveRounded'
import { api } from '../api/client'
import Alert from '../components/Alert'
import Card from '../components/Card'

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
    <Grid container spacing={3}>
      <Grid item xs={12} md={6}>
        <Card title="Rest Balance" subtitle="Manage rest minutes" icon={<LocalHotelOutlinedIcon />}>
          {error && <Alert type="error">{error}</Alert>}
          {msg && <Alert type="success">{msg}</Alert>}
          <Stack spacing={3}>
            <Typography variant="h3">{rest ?? '-'} min</Typography>
            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} alignItems="flex-start">
              <TextField
                label="Minutes"
                value={value}
                onChange={(e) => setValue(e.target.value)}
                inputMode="numeric"
                InputProps={{ endAdornment: <InputAdornment position="end">min</InputAdornment> }}
                sx={{ maxWidth: 200 }}
              />
              <Button variant="contained" startIcon={<AddRoundedIcon />} onClick={() => act('add')}>
                Add
              </Button>
              <Button variant="outlined" startIcon={<RemoveRoundedIcon />} onClick={() => act('spend')}>
                Spend
              </Button>
            </Stack>
          </Stack>
        </Card>
      </Grid>
    </Grid>
  )
}
