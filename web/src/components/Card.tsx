import { Card as MuiCard, CardActions, CardContent, CardHeader, Divider, Stack } from '@mui/material'
import Box from '@mui/material/Box'
import { PropsWithChildren, ReactNode } from 'react'

interface Props {
  title?: string
  subtitle?: string
  actions?: ReactNode
  icon?: ReactNode
}

export default function Card({ title, subtitle, actions, icon, children }: PropsWithChildren<Props>) {
  const header = title ? (
    <CardHeader
      title={title}
      subheader={subtitle}
      avatar={icon ? (
        <Box
          sx={{
            width: 44,
            height: 44,
            borderRadius: 2,
            display: 'grid',
            placeItems: 'center',
            bgcolor: 'rgba(99, 102, 241, 0.16)',
            color: 'primary.main',
          }}
        >
          {icon}
        </Box>
      ) : undefined}
      sx={{
        pb: 0,
        '& .MuiCardHeader-title': { fontWeight: 600 },
        '& .MuiCardHeader-subheader': { color: 'text.secondary' },
      }}
    />
  ) : null

  return (
    <MuiCard
      elevation={3}
      sx={{
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        position: 'relative',
        overflow: 'hidden',
        '&:before': icon
          ? {
              content: '""',
              position: 'absolute',
              inset: 0,
              pointerEvents: 'none',
              background:
                'radial-gradient(circle at top right, rgba(99, 102, 241, 0.18), transparent 45%)',
              opacity: 0.9,
            }
          : undefined,
      }}
    >
      {header}
      <CardContent sx={{ flexGrow: 1, position: 'relative' }}>{children}</CardContent>
      {actions && (
        <>
          <Divider sx={{ opacity: 0.5 }} />
          <CardActions sx={{ justifyContent: 'flex-end' }}>
            <Stack direction="row" spacing={1}>
              {actions}
            </Stack>
          </CardActions>
        </>
      )}
    </MuiCard>
  )
}
