export { LegalSheet } from './components/LegalSheet';
export { LegalFooter } from './components/LegalFooter';
export { getLegalContent } from './content';
export type { LegalPageId, LegalPageText, LegalSection } from './content';

import { LegalSheet } from './components/LegalSheet';
export const legalSheetMap = {
  legal: LegalSheet,
} as const;
