export { LegalSheet } from './components/LegalSheet';
export { LegalFooter } from './components/LegalFooter';
export { LEGAL_CONTENT } from './content';
export type { LegalPageId, LegalPageText, LegalSection } from './content';

import { LegalSheet } from './components/LegalSheet';
export const legalSheetMap = {
  legal: LegalSheet,
} as const;
