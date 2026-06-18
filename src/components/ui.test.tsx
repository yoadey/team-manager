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
    state: { primaryColor: '#4285F4', form: { title: 'Hello' } },
    onFormInput: vi.fn(),
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
  it('renders input with name from form state', () => {
    const { container } = render(<TextInput name="title" />);
    const input = container.querySelector('input[name="title"]');
    expect(input).toBeTruthy();
    expect((input as HTMLInputElement).value).toBe('Hello');
  });
});

describe('TextArea', () => {
  it('renders textarea element', () => {
    const { container } = render(<TextArea name="title" placeholder="Beschreibung" />);
    const ta = container.querySelector('textarea');
    expect(ta).toBeTruthy();
    expect(ta!.value).toBe('Hello');
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
