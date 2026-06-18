import useMediaQuery from '@mui/material/useMediaQuery';

export const COMPACT_BP = 760;
export function useCompact() {
  return useMediaQuery(`(max-width:${COMPACT_BP - 1}px)`);
}
