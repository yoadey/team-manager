export type { NewsItem } from './types';
export { NewsPage } from './NewsPage';
export { NewsFormSheet } from './components/NewsFormSheet';
export { useNewsActions } from './hooks/useNewsActions';
export { useNewsQuery } from './hooks/useNewsQueries';

import { NewsFormSheet } from './components/NewsFormSheet';
export const newsSheetMap = {
  newsForm: NewsFormSheet,
} as const;
