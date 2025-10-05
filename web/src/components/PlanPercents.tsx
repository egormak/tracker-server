import { useEffect, useState } from 'react'
import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import CircularProgress from '@mui/material/CircularProgress'
import Grid from '@mui/material/Grid'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import PieChartRoundedIcon from '@mui/icons-material/PieChartRounded'
import { api, PlanPercentGroup, PlanPercentsResponse } from '../api/client'
import Alert from './Alert'
import Card from './Card'
import Progress from './Progress'

export default function PlanPercents() {
  const [percents, setPercents] = useState<PlanPercentsResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [removing, setRemoving] = useState<string | null>(null)

  const loadPercents = async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await api.getPlanPercents()
      setPercents(data)
    } catch (e: any) {
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadPercents()
  }, [])

  const handleRemove = async (group: PlanPercentGroup, value: number) => {
    setError(null)
    setSuccess(null)
    const key = `${group}-${value}`
    setRemoving(key)
    try {
      await api.removePlanPercent(group, value)
      setSuccess(`Removed ${value}% from ${group}.`)
      await loadPercents()
    } catch (e: any) {
      setError(e.message)
    } finally {
      setRemoving(null)
    }
  }

  const renderPercentList = (group: PlanPercentGroup, title: string, values: number[]) => (
    <Stack spacing={1}>
      <Typography variant="h6" fontWeight={600}>
        {title}
      </Typography>
      <Stack direction="row" spacing={1} flexWrap="wrap">
        {values.length > 0 ? (
          values.map((value, index) => {
            const chipKey = `${group}-${value}`
            return (
            <Chip
              key={index}
              label={`${value}%`}
              size="small"
              color={index === 0 ? 'primary' : 'default'}
              variant={index === 0 ? 'filled' : 'outlined'}
              onDelete={() => handleRemove(group, value)}
              deleteIcon={removing === chipKey ? <CircularProgress size={14} /> : undefined}
              disabled={loading || removing === chipKey}
            />
            )
          })
        ) : (
          <Typography variant="body2" color="text.secondary">
            No percents set
          </Typography>
        )}
      </Stack>
    </Stack>
  )

  return (
    <Card title="Plan Percents" subtitle="View percents for plan, work, learn and rest" icon={<PieChartRoundedIcon />}>
      {error && <Alert type="error">{error}</Alert>}
      {success && <Alert type="success">{success}</Alert>}
      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', p: 3 }}>
          <CircularProgress />
        </Box>
      ) : percents ? (
        <Stack spacing={3}>
          <Box>
            <Typography variant="body1" gutterBottom>
              Current Choice: <strong>{percents.data.plans[percents.data.current_choice]}</strong>
            </Typography>
            <Typography variant="body2" color="text.secondary">
              Date: {percents.data.date}
            </Typography>
          </Box>
          
          <Grid container spacing={2}>
            <Grid item xs={12} sm={6}>
              {renderPercentList('plan', 'Plan', percents.data.plan)}
            </Grid>
            <Grid item xs={12} sm={6}>
              {renderPercentList('work', 'Work', percents.data.work)}
            </Grid>
            <Grid item xs={12} sm={6}>
              {renderPercentList('learn', 'Learn', percents.data.learn)}
            </Grid>
            <Grid item xs={12} sm={6}>
              {renderPercentList('rest', 'Rest', percents.data.rest)}
            </Grid>
          </Grid>

          {/* Progress visualization */}
          <Box sx={{ mt: 2 }}>
            <Typography variant="subtitle1" fontWeight={600} gutterBottom>
              Current Percent Distributions
            </Typography>
            <Stack spacing={2}>
              {percents.data.plan.length > 0 && (
                <Box>
                  <Typography variant="body2">Plan: {percents.data.plan[0]}%</Typography>
                  <Progress value={percents.data.plan[0]} />
                </Box>
              )}
              {percents.data.work.length > 0 && (
                <Box>
                  <Typography variant="body2">Work: {percents.data.work[0]}%</Typography>
                  <Progress value={percents.data.work[0]} />
                </Box>
              )}
              {percents.data.learn.length > 0 && (
                <Box>
                  <Typography variant="body2">Learn: {percents.data.learn[0]}%</Typography>
                  <Progress value={percents.data.learn[0]} />
                </Box>
              )}
              {percents.data.rest.length > 0 && (
                <Box>
                  <Typography variant="body2">Rest: {percents.data.rest[0]}%</Typography>
                  <Progress value={percents.data.rest[0]} />
                </Box>
              )}
            </Stack>
          </Box>
        </Stack>
      ) : (
        <Typography color="text.secondary">No plan percent data available.</Typography>
      )}
    </Card>
  )
}
