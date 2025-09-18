export default function Alert({ type, children }: { type: 'error' | 'success'; children: React.ReactNode }) {
  return <div className={`alert ${type}`}>{children}</div>
}

