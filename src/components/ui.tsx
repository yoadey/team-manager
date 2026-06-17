/* Shared UI atoms — faithful MUI re-implementations of the prototype's helper
   render functions (icon, avatar, chip, section title, buttons, form fields). */
import React from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { type SxProps, type Theme } from '@mui/material/styles';
import { initials as toInitials, NEUTRAL } from '../styles/tokens';
import { useApp } from '../store/AppContext';

/** Material Symbols Outlined glyph (rendered by glyph name, like the prototype). */
export function Sym({ name, size = 20, color = 'inherit', sx }: { name: string; size?: number; color?: string; sx?: SxProps<Theme> }) {
  return (
    <Box
      component="span"
      sx={{ fontFamily: "'Material Symbols Outlined'", fontSize: size + 'px', lineHeight: 1, color, flex: '0 0 auto', userSelect: 'none', ...(sx as object) }}
    >
      {name}
    </Box>
  );
}

/** Round avatar with photo or coloured initials. */
export function Av({ name, photo, color, size = 40, font }: { name?: string; photo?: string | null; color?: string; size?: number; font?: number }) {
  const f = font || Math.round(size * 0.36);
  const base: SxProps<Theme> = {
    width: size, height: size, borderRadius: '50%', flex: '0 0 auto', display: 'flex',
    alignItems: 'center', justifyContent: 'center', color: '#fff', fontWeight: 700, fontSize: f + 'px',
    overflow: 'hidden', backgroundColor: color || '#888',
  };
  if (photo) {
    return <Box component="span" sx={{ ...(base as object), backgroundImage: `url(${photo})`, backgroundSize: 'cover', backgroundPosition: 'center' }} />;
  }
  return <Box component="span" sx={base}>{toInitials(name || '')}</Box>;
}

/** Pill chip (status / type / label). */
export function Chip({ label, color, bg, icon, fs = 11 }: { label: React.ReactNode; color: string; bg: string; icon?: string; fs?: number }) {
  return (
    <Box component="span" sx={{ display: 'inline-flex', alignItems: 'center', gap: '4px', fontSize: fs + 'px', fontWeight: 700, color, background: bg, padding: '4px 9px', borderRadius: '999px', whiteSpace: 'nowrap' }}>
      {icon ? <Sym name={icon} size={fs + 3} color={color} /> : null}
      {label}
    </Box>
  );
}

export function SectionTitle({ children, right }: { children: React.ReactNode; right?: React.ReactNode }) {
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: '8px', m: '4px 2px 10px' }}>
      <Box sx={{ fontSize: '12px', fontWeight: 700, color: NEUTRAL.secondary, letterSpacing: '.4px', textTransform: 'uppercase', flex: 1 }}>{children}</Box>
      {right || null}
    </Box>
  );
}

export function EmptyState({ icon, text }: { icon: string; text: string }) {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: '10px', p: '56px 20px', color: NEUTRAL.faint }}>
      <Sym name={icon} size={46} />
      <Box sx={{ fontSize: '14px', textAlign: 'center' }}>{text}</Box>
    </Box>
  );
}

export function SpinnerBox() {
  const { state } = useApp();
  const { primaryColor } = state;
  return (
    <Box sx={{ display: 'flex', justifyContent: 'center', p: '48px' }}>
      <Box sx={{ width: 34, height: 34, border: '3px solid #D5D8E0', borderTopColor: primaryColor, borderRadius: '50%', animation: 'tvSpin .8s linear infinite' }} />
    </Box>
  );
}

export function Spinner({ size = 16, color = 'currentColor' }: { size?: number; color?: string }) {
  return <Box component="span" sx={{ width: size, height: size, border: `2px solid ${color}`, borderTopColor: 'transparent', borderRadius: '50%', display: 'inline-block', animation: 'tvSpin .7s linear infinite' }} />;
}

export function Card({ children, sx }: { children: React.ReactNode; sx?: SxProps<Theme> }) {
  return <Box sx={{ background: '#fff', border: `1px solid ${NEUTRAL.line}`, borderRadius: '18px', p: '16px', ...(sx as object) }}>{children}</Box>;
}

/** Full-width primary button matching the prototype's primaryBtn(). */
export function PrimaryButton({ label, onClick, disabled, busy }: { label: React.ReactNode; onClick?: () => void; disabled?: boolean; busy?: boolean }) {
  const theme = useApp().state.primaryColor;
  return (
    <ButtonBase
      onClick={onClick}
      disabled={disabled || busy}
      sx={{
        display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '9px', width: '100%', p: '14px',
        borderRadius: '14px', border: 'none', cursor: disabled ? 'default' : 'pointer',
        background: (disabled || busy) ? NEUTRAL.inputBorder : theme, color: '#fff', fontSize: '15px', fontWeight: 600, mt: '4px',
      }}
    >
      {busy ? <Spinner /> : null}
      {label}
    </ButtonBase>
  );
}

export const labelSx: SxProps<Theme> = { fontSize: '12px', fontWeight: 600, color: NEUTRAL.onSurfaceVariant, mb: '6px' };
export const inputSx: React.CSSProperties = { width: '100%', border: `1.5px solid ${NEUTRAL.inputBorder}`, borderRadius: '13px', padding: '12px 14px', fontSize: '14px', outline: 'none', background: '#fff', color: NEUTRAL.onSurface, fontFamily: 'inherit' };

export function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <Box component="label" sx={{ display: 'block' }}>
      <Box sx={labelSx}>{label}</Box>
      {children}
    </Box>
  );
}

/** Form-bound text input (mirrors prototype tf()). */
export function TextInput({ name, type = 'text', placeholder, min, max, ...rest }: { name: string; type?: string; placeholder?: string; min?: string; max?: string; [k: string]: any }) {
  const { state, onFormInput } = useApp();
  const v = state.form[name];
  return (
    <input
      name={name}
      type={type}
      min={min}
      max={max}
      value={v == null ? '' : v}
      placeholder={placeholder || ''}
      onChange={onFormInput}
      style={inputSx}
      {...rest}
    />
  );
}

export function TextArea({ name, placeholder, minHeight = 80 }: { name: string; placeholder?: string; minHeight?: number }) {
  const { state, onFormInput } = useApp();
  const v = state.form[name];
  return (
    <textarea
      name={name}
      value={v == null ? '' : v}
      placeholder={placeholder || ''}
      onChange={onFormInput}
      style={{ ...inputSx, minHeight, resize: 'vertical' }}
    />
  );
}

/** Small square icon button used in lists. */
export function IconBtn({ icon, onClick, color = NEUTRAL.secondary, bg = NEUTRAL.sidebar, title, size = 30 }: { icon: string; onClick?: () => void; color?: string; bg?: string; title?: string; size?: number }) {
  return (
    <ButtonBase title={title} onClick={onClick} sx={{ width: size, height: size, borderRadius: '8px', background: bg, color, flex: '0 0 auto' }}>
      <Sym name={icon} size={17} color={color} />
    </ButtonBase>
  );
}

export function metaItem(icon: string, text: React.ReactNode, key?: string) {
  return (
    <Box key={key} component="span" sx={{ display: 'inline-flex', alignItems: 'center', gap: '4px', whiteSpace: 'nowrap', maxWidth: '180px', overflow: 'hidden', textOverflow: 'ellipsis' }}>
      <Sym name={icon} size={15} />
      {text}
    </Box>
  );
}
