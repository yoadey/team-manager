import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { buildTokens } from '@/styles/tokens';
import { Field, PrimaryButton, Sym, TextArea, TextInput } from '@/components/ui';
import type { SheetProps } from '@/sheets/types';

export function NewsFormSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  void sheet;
  const F = app.state.form;
  const pin = (
    <ButtonBase key="pin" onClick={() => app.setFormVal({ pinned: !F.pinned })} sx={{ display: 'flex', alignItems: 'center', gap: '12px', width: '100%', p: '12px 14px', borderRadius: '13px', cursor: 'pointer', border: '1px solid #E6E7EE', background: '#F4F4FA' }}>
      <Sym name="push_pin" size={20} color={F.pinned ? t.primary : '#9A9DA6'} />
      <Box component="span" sx={{ flex: 1, textAlign: 'left', fontSize: '14px', fontWeight: 500 }}>Oben anpinnen</Box>
      <Box component="span" sx={{ width: '44px', height: '26px', borderRadius: '999px', background: F.pinned ? t.primary : '#C8CAD2', position: 'relative' }}>
        <Box component="span" sx={{ position: 'absolute', top: '3px', left: F.pinned ? '21px' : '3px', width: '20px', height: '20px', borderRadius: '50%', background: '#fff', transition: 'left .2s' }} />
      </Box>
    </ButtonBase>
  );
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Field label="Titel"><TextInput name="title" placeholder="Überschrift" /></Field>
      <Field label="Text"><TextArea name="body" placeholder="Was gibt es Neues?" minHeight={120} /></Field>
      {pin}
      <PrimaryButton label="Veröffentlichen" onClick={() => app.saveNews()} busy={app.state.busy === 'save'} />
    </Box>
  );
}
