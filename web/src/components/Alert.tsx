import MuiAlert from '@mui/material/Alert'

export default function Alert({ type, children }: { type: 'error' | 'success'; children: React.ReactNode }) {
  return (
    <MuiAlert severity={type === 'error' ? 'error' : 'success'} variant="filled" sx={{ mb: 2 }}>
      {children}
    </MuiAlert>
  )
}
