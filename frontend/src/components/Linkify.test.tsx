import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Linkify } from './Linkify';

describe('Linkify', () => {
  it('renders plain text without links unchanged', () => {
    render(<Linkify text="Kein Link hier." />);
    expect(screen.getByText('Kein Link hier.')).toBeTruthy();
    expect(screen.queryByRole('link')).toBeNull();
  });

  it('turns a bare https URL into a clickable link', () => {
    render(<Linkify text="Details unter https://example.com/anmeldung finden sich hier." />);
    const link = screen.getByRole('link', { name: 'https://example.com/anmeldung' });
    expect(link.getAttribute('href')).toBe('https://example.com/anmeldung');
    expect(link.getAttribute('target')).toBe('_blank');
    expect(link.getAttribute('rel')).toBe('noopener noreferrer');
  });

  it('links a www.-prefixed URL, prefixing https:// in the href but keeping the displayed text', () => {
    render(<Linkify text="Siehe www.example.com für mehr Infos." />);
    const link = screen.getByRole('link', { name: 'www.example.com' });
    expect(link.getAttribute('href')).toBe('https://www.example.com');
  });

  it('links multiple URLs in the same text', () => {
    render(<Linkify text="Erst https://a.example.com dann https://b.example.com" />);
    expect(screen.getAllByRole('link')).toHaveLength(2);
  });
});
