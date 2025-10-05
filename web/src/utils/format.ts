export function formatRestMinutes(raw: number | null | undefined): string {
  if (raw == null) return '-'
  const minutes = raw / 100
  const hasFraction = Math.abs(minutes - Math.round(minutes)) > Number.EPSILON
  return new Intl.NumberFormat(undefined, {
    minimumFractionDigits: hasFraction ? 1 : 0,
    maximumFractionDigits: 2,
  }).format(minutes)
}
