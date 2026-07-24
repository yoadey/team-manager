import { useSyncExternalStore } from 'react';
import Box from '@mui/material/Box';
import type { SheetProps } from '@/sheets/types';
import { NEUTRAL } from '@/styles/tokens';
import { getLocale, subscribeLocale } from '@/i18n';
import { LEGAL_CONTENT } from '../content';

// Subscribes to the module-level i18n store directly (same pattern as
// layouts/AppShell.tsx and components/SheetHost.tsx) so a locale switch while
// this sheet is open re-renders it with the other language's content.
function useLocaleSubscription(): void {
  useSyncExternalStore(subscribeLocale, getLocale);
}

/** Renders the static legal-notice or privacy-policy content (see ../content.ts). */
export function LegalSheet({ sheet }: SheetProps) {
  useLocaleSubscription();
  const page = sheet.legalPage ?? 'impressum';
  const content = LEGAL_CONTENT[getLocale()][page];

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
      {content.sections.map((section) => (
        <Box key={section.heading}>
          <Box component="h3" sx={{ fontSize: '14px', fontWeight: 700, color: NEUTRAL.onSurface, m: '0 0 8px' }}>
            {section.heading}
          </Box>
          {section.paragraphs.map((p, i) => (
            <Box key={i} component="p" sx={{ fontSize: '13px', color: NEUTRAL.secondary, lineHeight: 1.6, m: '0 0 8px' }}>
              {p}
            </Box>
          ))}
          {section.list ? (
            <Box component="ul" sx={{ m: '0 0 8px', pl: '20px' }}>
              {section.list.map((item, i) => (
                <Box
                  key={i}
                  component="li"
                  sx={{ fontSize: '13px', color: NEUTRAL.secondary, lineHeight: 1.6, mb: '4px' }}
                >
                  {item}
                </Box>
              ))}
            </Box>
          ) : null}
        </Box>
      ))}
    </Box>
  );
}
