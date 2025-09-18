export default function Progress({ value }: { value: number }) {
  const v = Math.max(0, Math.min(100, Math.round(value)))
  return (
    <div className="progress" aria-valuemin={0} aria-valuemax={100} aria-valuenow={v} role="progressbar">
      <span style={{ width: `${v}%` }} />
    </div>
  )
}

