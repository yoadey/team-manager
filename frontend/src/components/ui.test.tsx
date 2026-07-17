import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {
  Sym,
  Av,
  Chip,
  SectionTitle,
  EmptyState,
  Spinner,
  Card,
  PrimaryButton,
  Field,
  TextInput,
  TextArea,
  IconBtn,
  metaItem,
} from './ui';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn().mockReturnValue({
    state: { primaryColor: '#4285F4' },
  }),
}));

describe('Sym', () => {
  it('renders icon name as text', () => {
    render(<Sym name="home" />);
    expect(screen.getByText('home')).toBeTruthy();
  });

  it('exposes label to assistive tech when label provided', () => {
    render(<Sym name="home" label="Startseite" />);
    expect(screen.getByRole('img', { name: 'Startseite' })).toBeTruthy();
  });
});

describe('Av', () => {
  it('renders initials when no photo', () => {
    render(<Av name="Anna Müller" color="#4285F4" />);
    expect(screen.getByText('AM')).toBeTruthy();
  });

  it('renders photo when provided', () => {
    const { container } = render(<Av name="Anna" photo="data:image/png;base64,abc" color="#4285F4" />);
    expect(container.firstChild).toBeTruthy();
  });
});

describe('Chip', () => {
  it('renders label text', () => {
    render(<Chip label="Active" color="#000" bg="#fff" />);
    expect(screen.getByText('Active')).toBeTruthy();
  });

  it('renders icon when provided', () => {
    render(<Chip label="Active" color="#000" bg="#fff" icon="check" />);
    expect(screen.getByText('check')).toBeTruthy();
  });
});

describe('SectionTitle', () => {
  it('renders children', () => {
    render(<SectionTitle>Meine Termine</SectionTitle>);
    expect(screen.getByText('Meine Termine')).toBeTruthy();
  });

  it('renders right slot', () => {
    render(<SectionTitle right={<button>Action</button>}>Title</SectionTitle>);
    expect(screen.getByText('Action')).toBeTruthy();
  });
});

describe('EmptyState', () => {
  it('renders empty state text', () => {
    render(<EmptyState icon="event" text="Keine Termine" />);
    expect(screen.getByText('Keine Termine')).toBeTruthy();
  });
});

describe('Spinner', () => {
  it('renders without crashing', () => {
    render(<Spinner />);
    expect(screen.getByRole('status')).toBeTruthy();
  });

  it('uses custom size and color', () => {
    const { container } = render(<Spinner size={32} color="#FF0000" />);
    expect(container.firstChild).toBeTruthy();
  });
});

describe('Card', () => {
  it('renders children', () => {
    render(<Card>Card content</Card>);
    expect(screen.getByText('Card content')).toBeTruthy();
  });
});

describe('PrimaryButton', () => {
  it('renders label', () => {
    render(<PrimaryButton label="Speichern" />);
    expect(screen.getByText('Speichern')).toBeTruthy();
  });

  it('calls onClick when clicked', async () => {
    const onClick = vi.fn();
    render(<PrimaryButton label="Speichern" onClick={onClick} />);
    await userEvent.click(screen.getByText('Speichern'));
    expect(onClick).toHaveBeenCalled();
  });

  it('shows spinner when busy', () => {
    render(<PrimaryButton label="Speichern" busy />);
    expect(screen.getByRole('status')).toBeTruthy();
  });
});

describe('Field', () => {
  it('renders label', () => {
    render(
      <Field label="Titel">
        <input type="text" />
      </Field>,
    );
    expect(screen.getByText('Titel')).toBeTruthy();
  });

  it('renders required asterisk when required=true', () => {
    render(
      <Field label="Titel" required>
        <input type="text" />
      </Field>,
    );
    expect(screen.getByText('*')).toBeTruthy();
  });

  it('renders error text when error+errorText provided', () => {
    render(
      <Field label="Titel" error errorText="Pflichtfeld">
        <input type="text" />
      </Field>,
    );
    expect(screen.getByText('Pflichtfeld')).toBeTruthy();
    expect(screen.getByRole('alert')).toBeTruthy();
  });

  it('renders helper text when provided', () => {
    render(
      <Field label="Titel" helperText="Hilfetext">
        <input type="text" />
      </Field>,
    );
    expect(screen.getByText('Hilfetext')).toBeTruthy();
  });
});

describe('TextInput', () => {
  it('renders a controlled input reflecting the passed value', () => {
    const { container } = render(<TextInput name="title" value="Hello" onChange={() => {}} />);
    const input = container.querySelector('input[name="title"]');
    expect(input).toBeTruthy();
    expect((input as HTMLInputElement).value).toBe('Hello');
  });
});

describe('TextArea', () => {
  it('renders a controlled textarea reflecting the passed value', () => {
    const { container } = render(
      <TextArea name="title" placeholder="Beschreibung" value="Hello" onChange={() => {}} />,
    );
    const ta = container.querySelector('textarea');
    expect(ta).toBeTruthy();
    expect(ta!.value).toBe('Hello');
  });

  // Regression: TextArea didn't forward extra props (no ...rest), so
  // Field's cloneElement-injected aria-invalid/aria-required/aria-describedby
  // were silently dropped -- the red-border style still applied (it's
  // explicitly destructured), so the field visually looked broken to
  // sighted users while screen readers got no indication anything was wrong.
  it('forwards aria attributes injected by a wrapping Field', () => {
    const { container } = render(
      <Field label="Beschreibung" required error errorText="Pflichtfeld">
        <TextArea name="title" />
      </Field>,
    );
    const ta = container.querySelector('textarea');
    expect(ta).toBeTruthy();
    expect(ta!.getAttribute('aria-required')).toBe('true');
    expect(ta!.getAttribute('aria-invalid')).toBe('true');
    expect(ta!.getAttribute('aria-describedby')).toBeTruthy();
  });
});

describe('IconBtn', () => {
  it('renders without crashing', () => {
    const { container } = render(<IconBtn icon="edit" title="Bearbeiten" />);
    expect(container.firstChild).toBeTruthy();
  });

  it('calls onClick when clicked', async () => {
    const onClick = vi.fn();
    render(<IconBtn icon="edit" onClick={onClick} title="Bearbeiten" />);
    await userEvent.click(screen.getByRole('button'));
    expect(onClick).toHaveBeenCalled();
  });
});

describe('metaItem', () => {
  it('renders text content', () => {
    render(<>{metaItem('location_on', 'Berlin')}</>);
    expect(screen.getByText('Berlin')).toBeTruthy();
  });
});
