import useMediaQuery from '@mui/material/useMediaQuery';

export const COMPACT_BP = 760;
export function useCompact() {
  return useMediaQuery(`(max-width:${COMPACT_BP - 1}px)`);
}

// Shortens a team name for compact display (team-switcher header, its
// aria-label, team-settings title, invite-description highlight): returns
// just the first word, e.g. "SG Muster" -> "SG", "A-Team TSC Schwarz-Gelb
// Aachen" -> "A-Team". A name with no whitespace is returned unchanged --
// it's already as short as this can make it.
export function shortName(name: string) {
  return name.trim().split(/\s+/)[0];
}
