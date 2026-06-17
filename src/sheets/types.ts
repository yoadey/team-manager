import type { AppContextValue, SheetState } from '../store/AppContext';

export interface SheetProps {
  app: AppContextValue;
  sheet: SheetState;
}
