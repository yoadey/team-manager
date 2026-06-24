import type { AppContextValue, SheetState } from '@/context/AppContext';

export interface SheetProps {
  app: AppContextValue;
  sheet: SheetState;
}
