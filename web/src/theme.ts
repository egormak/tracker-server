import { createTheme, responsiveFontSizes } from '@mui/material/styles'

const baseTheme = createTheme({
  palette: {
    mode: 'dark',
    primary: { main: '#6366f1' },
    secondary: { main: '#22d3ee' },
    background: { default: '#05070d', paper: 'rgba(17, 24, 39, 0.82)' },
  },
  shape: { borderRadius: 16 },
  typography: {
    fontFamily: '"Roboto", "Inter", "Helvetica", "Arial", sans-serif',
    h3: { fontWeight: 700, letterSpacing: '-0.02em' },
    h4: { fontWeight: 700 },
    h5: { fontWeight: 600 },
    subtitle1: { fontWeight: 500 },
  },
  components: {
    MuiCssBaseline: {
      styleOverrides: {
        body: {
          backgroundImage:
            'radial-gradient(circle at 20% 20%, rgba(99, 102, 241, 0.18), transparent 55%), radial-gradient(circle at 80% 0%, rgba(34, 211, 238, 0.12), transparent 45%), linear-gradient(180deg, #020308 0%, #05070d 100%)',
        },
      },
    },
    MuiContainer: {
      defaultProps: { maxWidth: 'lg' },
    },
    MuiAppBar: {
      styleOverrides: {
        root: {
          backgroundImage:
            'linear-gradient(135deg, rgba(15, 23, 42, 0.92), rgba(15, 23, 42, 0.75))',
          boxShadow: '0 18px 40px rgba(15, 23, 42, 0.28)',
          borderBottom: '1px solid rgba(148, 163, 184, 0.12)',
        },
      },
    },
    MuiCard: {
      styleOverrides: {
        root: {
          backgroundImage:
            'linear-gradient(180deg, rgba(17, 24, 39, 0.92) 0%, rgba(15, 23, 42, 0.88) 100%)',
          border: '1px solid rgba(148, 163, 184, 0.14)',
          boxShadow: '0 24px 55px rgba(15, 23, 42, 0.45)',
          backdropFilter: 'blur(22px)',
        },
      },
    },
    MuiButton: {
      styleOverrides: {
        root: {
          borderRadius: 999,
          textTransform: 'none',
          fontWeight: 600,
          paddingInline: 18,
        },
        containedPrimary: {
          backgroundImage: 'linear-gradient(135deg, #6366f1, #2563eb)',
          boxShadow: '0 12px 24px rgba(99, 102, 241, 0.28)',
        },
        outlined: {
          borderColor: 'rgba(99, 102, 241, 0.4)',
        },
      },
    },
    MuiList: {
      styleOverrides: { root: { paddingTop: 0, paddingBottom: 0 } },
    },
    MuiListItem: {
      styleOverrides: {
        root: {
          borderRadius: 14,
          padding: '16px 20px',
          marginBottom: 12,
          border: '1px solid rgba(148, 163, 184, 0.1)',
          backgroundColor: 'rgba(8, 12, 20, 0.6)',
        },
      },
    },
    MuiDivider: {
      styleOverrides: { root: { borderColor: 'rgba(148, 163, 184, 0.18)' } },
    },
  },
})

const theme = responsiveFontSizes(baseTheme)

export default theme
