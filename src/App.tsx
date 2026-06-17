import { useMemo } from 'react';
import { ThemeProvider } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';
import { AppProvider, useApp } from './context/AppContext';
import { buildMuiTheme } from './styles/theme';
import { Root } from './components/Root';

function Themed() {
  const { state } = useApp();
  const theme = useMemo(() => buildMuiTheme(state.primaryColor), [state.primaryColor]);
  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Root />
    </ThemeProvider>
  );
}

export function App() {
  return (
    <AppProvider>
      <Themed />
    </AppProvider>
  );
}
