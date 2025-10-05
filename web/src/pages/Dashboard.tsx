import { useEffect, useMemo, useState, type ReactNode } from 'react'
import Avatar from '@mui/material/Avatar'
import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import Grid from '@mui/material/Grid'
import List from '@mui/material/List'
import ListItem from '@mui/material/ListItem'
import ListItemAvatar from '@mui/material/ListItemAvatar'
import ListItemText from '@mui/material/ListItemText'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import AccessTimeOutlinedIcon from '@mui/icons-material/AccessTimeOutlined'
import HistoryToggleOffOutlinedIcon from '@mui/icons-material/HistoryToggleOffOutlined'
import TimelineOutlinedIcon from '@mui/icons-material/TimelineOutlined'
import NightlightRoundOutlinedIcon from '@mui/icons-material/NightlightRoundOutlined'
import TaskAltOutlinedIcon from '@mui/icons-material/TaskAltOutlined'
import CategoryOutlinedIcon from '@mui/icons-material/CategoryOutlined'
import { api, RestTimeResponse, RecordsSummary, TaskResult } from '../api/client'
import Alert from '../components/Alert'
import Card from '../components/Card'
import Progress from '../components/Progress'
import { formatRestMinutes } from '../utils/format'

export default function Dashboard() {
  const [records, setRecords] = useState<RecordsSummary | null>(null)
  const [rest, setRest] = useState<RestTimeResponse | null>(null)
  const [tasks, setTasks] = useState<TaskResult[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let mounted = true
    async function load() {
      try {
        const [rec, r, tl] = await Promise.all([
          api.getRecordsSummary(),
          api.restGet(),
          api.getStatsTasksToday(),
        ])
        if (!mounted) return
        setRecords(rec)
        setRest(r)
        setTasks(tl)
      } catch (e: any) {
        setError(e.message)
      } finally {
        setLoading(false)
      }
    }
    load()
    return () => {
      mounted = false
    }
  }, [])

  const totals = useMemo(() => {
    const sum = (m?: Record<string, number>) => Object.values(m || {}).reduce((a, b) => a + b, 0)
    const today = sum(records?.today)
    const yesterday = sum(records?.yesterday)
    const all = sum(records?.all)
    const vsYesterday = yesterday ? Math.round((today / yesterday) * 100) : 0
    return { today, yesterday, all, vsYesterday }
  }, [records])

  const overviewItems = [
    {
      label: 'Total Today',
      value: `${totals.today} min`,
      icon: <AccessTimeOutlinedIcon fontSize="small" />,
    },
    {
      label: 'Yesterday',
      value: `${totals.yesterday} min`,
      icon: <HistoryToggleOffOutlinedIcon fontSize="small" />,
      align: 'flex-end' as const,
    },
    {
      label: 'All Time',
      value: `${totals.all} min`,
      icon: <TimelineOutlinedIcon fontSize="small" />,
    },
    rest && {
      label: 'Rest available',
      value: `${formatRestMinutes(rest.rest_time)} min`,
      icon: <NightlightRoundOutlinedIcon fontSize="small" />,
    },
  ].filter(Boolean) as Array<{ label: string; value: string; icon: ReactNode; align?: 'flex-start' | 'flex-end' }>

  return (
    <Grid container spacing={3} alignItems="stretch">
      <Grid item xs={12} md={6}>
        <Card title="Overview" subtitle="Today summary" icon={<TaskAltOutlinedIcon />}>
          {error && <Alert type="error">{error}</Alert>}
          {loading ? (
            <Stack spacing={2.5}>
              {[1, 2, 3].map((key) => (
                <Skeleton key={key} variant="rounded" height={84} animation="wave" />
              ))}
            </Stack>
          ) : (
            <Stack spacing={3}>
              <Grid container spacing={2}>
                {overviewItems.map((item) => (
                  <Grid item xs={12} sm={6} key={item.label}>
                    <Stack
                      spacing={1}
                      alignItems={item.align || 'flex-start'}
                      sx={{
                        p: 2.2,
                        borderRadius: 3,
                        background: 'linear-gradient(135deg, rgba(99, 102, 241, 0.18), rgba(8, 20, 38, 0.4))',
                        border: '1px solid rgba(99, 102, 241, 0.22)',
                      }}
                    >
                      <Avatar
                        sx={{
                          width: 38,
                          height: 38,
                          bgcolor: 'rgba(99, 102, 241, 0.2)',
                          color: 'primary.main',
                        }}
                      >
                        {item.icon}
                      </Avatar>
                      <Box textAlign={item.align || 'left'}>
                        <Typography variant="overline" color="text.secondary" letterSpacing={0.6}>
                          {item.label}
                        </Typography>
                        <Typography variant="h5">{item.value}</Typography>
                      </Box>
                    </Stack>
                  </Grid>
                ))}
              </Grid>
              <Stack spacing={1.5}>
                <Typography variant="subtitle2" color="text.secondary">
                  Today vs Yesterday
                </Typography>
                <Tooltip title={`${totals.vsYesterday}% of yesterday`} placement="top">
                  <Box>
                    <Progress value={totals.vsYesterday} />
                  </Box>
                </Tooltip>
                <Chip
                  label={`${totals.vsYesterday}% of yesterday`}
                  size="small"
                  color={totals.vsYesterday >= 100 ? 'secondary' : 'default'}
                  sx={{ alignSelf: 'flex-start', bgcolor: 'rgba(99, 102, 241, 0.16)' }}
                />
              </Stack>
            </Stack>
          )}
        </Card>
      </Grid>

      <Grid item xs={12} md={6}>
        <Card title="Today Records" subtitle="Per category (aggregated)" icon={<CategoryOutlinedIcon />}>
          {loading && (
            <Stack spacing={2.5}>
              {[1, 2, 3].map((key) => (
                <Skeleton key={key} variant="rounded" height={68} animation="wave" />
              ))}
            </Stack>
          )}
          {!loading && (!records || Object.keys(records.today || {}).length === 0) && (
            <Typography color="text.secondary">No data</Typography>
          )}
          {!loading && records && (
            <List disablePadding>
              {Object.entries(records.today)
                .sort((a, b) => b[1] - a[1])
                .map(([name, minutes], index, arr) => (
                  <ListItem
                    key={name}
                    sx={{
                      mb: index === arr.length - 1 ? 0 : 1.5,
                      alignItems: 'center',
                      gap: 2,
                    }}
                  >
                    <ListItemAvatar>
                      <Avatar sx={{ bgcolor: 'rgba(99, 102, 241, 0.22)', color: 'primary.main' }}>
                        {name.slice(0, 1).toUpperCase()}
                      </Avatar>
                    </ListItemAvatar>
                    <ListItemText
                      primary={
                        <Typography variant="subtitle1" fontWeight={600} noWrap>
                          {name}
                        </Typography>
                      }
                    />
                    <Chip label={`${minutes} min`} color="secondary" variant="outlined" />
                  </ListItem>
                ))}
            </List>
          )}
        </Card>
      </Grid>

      <Grid item xs={12}>
        <Card title="Tasks by Plan" subtitle="Planned vs done (today)" icon={<TaskAltOutlinedIcon />}>
          {loading && (
            <Stack spacing={2.5}>
              {[1, 2, 3].map((key) => (
                <Skeleton key={key} variant="rounded" height={76} animation="wave" />
              ))}
            </Stack>
          )}
          {!loading && tasks.length === 0 && (
            <Typography color="text.secondary">No tasks</Typography>
          )}
          {!loading && tasks.length > 0 && (
            <List disablePadding>
              {tasks.map((t, index) => {
                const pct = t.time_duration ? (t.time_done / t.time_duration) * 100 : 0
                const intensity = Math.min(100, Math.max(0, pct)) / 100
                return (
                  <ListItem
                    key={t.name}
                    sx={{
                      mb: index === tasks.length - 1 ? 0 : 1.5,
                      alignItems: 'stretch',
                      gap: 2,
                    }}
                  >
                    <ListItemAvatar>
                      <Avatar
                        sx={{
                          bgcolor: `rgba(99, 102, 241, ${0.16 + intensity * 0.3})`,
                          color: 'primary.main',
                        }}
                      >
                        {t.role.slice(0, 1).toUpperCase()}
                      </Avatar>
                    </ListItemAvatar>
                    <ListItemText
                      primary={
                        <Typography variant="subtitle1" fontWeight={600}>
                          {t.name}
                        </Typography>
                      }
                      secondary={
                        <Stack direction="row" spacing={1} alignItems="center" flexWrap="wrap">
                          <Chip
                            size="small"
                            label={`priority ${t.priority}`}
                            variant="outlined"
                            sx={{ borderColor: 'rgba(99, 102, 241, 0.42)' }}
                          />
                          <Typography variant="body2" color="text.secondary">
                            role: {t.role}
                          </Typography>
                        </Stack>
                      }
                    />
                    <Stack spacing={1} alignItems="flex-end" sx={{ minWidth: 220 }}>
                      <Typography variant="subtitle1" fontWeight={600}>
                        {t.time_done} / {t.time_duration} min
                      </Typography>
                      <Chip
                        size="small"
                        label={`${Math.round(pct)}% done`}
                        color={pct >= 100 ? 'secondary' : 'default'}
                        sx={{ bgcolor: 'rgba(99, 102, 241, 0.16)' }}
                      />
                      <Tooltip title={`${Math.round(pct)}% complete`} placement="top">
                        <Box width="100%">
                          <Progress value={pct} />
                        </Box>
                      </Tooltip>
                    </Stack>
                  </ListItem>
                )
              })}
            </List>
          )}
        </Card>
      </Grid>
    </Grid>
  )
}
