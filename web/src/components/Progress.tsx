import LinearProgress from '@mui/material/LinearProgress'

export default function Progress({ value }: { value: number }) {
  const v = Math.max(0, Math.min(100, Math.round(value)))
  return <LinearProgress variant="determinate" value={v} aria-valuenow={v} />
}
