import AppBar from '@mui/material/AppBar'
import Avatar from '@mui/material/Avatar'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Container from '@mui/material/Container'
import Stack from '@mui/material/Stack'
import Toolbar from '@mui/material/Toolbar'
import Typography from '@mui/material/Typography'
import DashboardOutlinedIcon from '@mui/icons-material/DashboardOutlined'
import PlaylistAddCheckRoundedIcon from '@mui/icons-material/PlaylistAddCheckRounded'
import LocalHotelOutlinedIcon from '@mui/icons-material/LocalHotelOutlined'
import AssignmentTurnedInOutlinedIcon from '@mui/icons-material/AssignmentTurnedInOutlined'
import ManageAccountsRoundedIcon from '@mui/icons-material/ManageAccountsRounded'
import TimelapseOutlinedIcon from '@mui/icons-material/TimelapseOutlined'
import { ReactNode } from 'react'
import { Link as RouterLink, useMatch, useResolvedPath } from 'react-router-dom'

const navItems = [
  { to: '/', label: 'Dashboard', icon: <DashboardOutlinedIcon fontSize="small" />, end: true },
  { to: '/plan', label: 'Plan', icon: <PlaylistAddCheckRoundedIcon fontSize="small" /> },
  { to: '/rest', label: 'Rest', icon: <LocalHotelOutlinedIcon fontSize="small" /> },
  { to: '/record', label: 'Record', icon: <AssignmentTurnedInOutlinedIcon fontSize="small" /> },
  { to: '/manage', label: 'Manage', icon: <ManageAccountsRoundedIcon fontSize="small" /> },
  { to: '/timer', label: 'Timer', icon: <TimelapseOutlinedIcon fontSize="small" /> },
]

export default function Header() {
  return (
    <AppBar position="sticky" color="transparent" elevation={0} sx={{ backdropFilter: 'blur(12px)' }}>
      <Container>
        <Toolbar disableGutters sx={{ columnGap: 2 }}>
          <Stack direction="row" spacing={2} alignItems="center">
            <Avatar
              sx={{
                bgcolor: 'rgba(99, 102, 241, 0.22)',
                color: 'primary.main',
                width: 40,
                height: 40,
                fontWeight: 700,
                letterSpacing: '0.08em',
              }}
            >
              T
            </Avatar>
            <Box>
              <Typography variant="h6" component="div" sx={{ fontWeight: 700 }}>
                Tracker
              </Typography>
              <Typography variant="caption" color="rgba(226, 232, 240, 0.65)">
                Focused daily progress
              </Typography>
            </Box>
          </Stack>
          <Box sx={{ flexGrow: 1 }} />
          <Stack direction="row" spacing={1} component="nav">
            {navItems.map((item) => (
              <NavItemButton key={item.to} {...item} />
            ))}
          </Stack>
        </Toolbar>
      </Container>
    </AppBar>
  )
}

interface NavItemProps {
  to: string
  label: string
  icon?: ReactNode
  end?: boolean
}

function NavItemButton({ to, label, icon, end }: NavItemProps) {
  const resolved = useResolvedPath(to)
  const match = useMatch({ path: resolved.pathname, end })

  return (
    <Button
      component={RouterLink}
      to={to}
      color="inherit"
      size="large"
      sx={{
        textTransform: 'none',
        borderRadius: 999,
        color: match ? 'primary.main' : 'rgba(226, 232, 240, 0.76)',
        bgcolor: match
          ? 'rgba(99, 102, 241, 0.16)'
          : 'rgba(148, 163, 184, 0.08)',
        px: 2.5,
        gap: 1,
        '&:hover': {
          bgcolor: 'rgba(99, 102, 241, 0.22)',
        },
      }}
    >
      {icon}
      {label}
    </Button>
  )
}
