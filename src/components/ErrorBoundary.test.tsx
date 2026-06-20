import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ErrorBoundary, AppErrorFallback } from './ErrorBoundary';

function Boom(): never {
  throw new Error('explode');
}

describe('ErrorBoundary', () => {
  it('renders children when there is no error', () => {
    render(
      <ErrorBoundary>
        <div>safe content</div>
      </ErrorBoundary>,
    );
    expect(screen.getByText('safe content')).toBeInTheDocument();
  });

  it('renders the default fallback and reports the error', () => {
    const onError = vi.fn();
    render(
      <ErrorBoundary onError={onError}>
        <Boom />
      </ErrorBoundary>,
    );
    expect(screen.getByText('Etwas ist schiefgelaufen')).toBeInTheDocument();
    expect(onError).toHaveBeenCalledOnce();
    expect(onError.mock.calls[0][0]).toBeInstanceOf(Error);
  });

  it('supports a custom renderFallback that receives the error', () => {
    render(
      <ErrorBoundary renderFallback={(error) => <div>caught: {error.message}</div>}>
        <Boom />
      </ErrorBoundary>,
    );
    expect(screen.getByText('caught: explode')).toBeInTheDocument();
  });
});

describe('AppErrorFallback', () => {
  it('renders app-level error title', () => {
    render(<AppErrorFallback error={new Error('test error')} />);
    expect(screen.getByText('Die App konnte nicht geladen werden')).toBeInTheDocument();
  });

  it('renders app-level error subtitle', () => {
    render(<AppErrorFallback error={new Error('test error')} />);
    expect(screen.getByText('Bitte lade die Seite neu.')).toBeInTheDocument();
  });

  it('renders a reload button', () => {
    render(<AppErrorFallback error={new Error('test error')} />);
    const btn = document.querySelector('button');
    expect(btn).toBeTruthy();
  });

  it('clicking the reload button calls location.reload', () => {
    const reloadMock = vi.fn();
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { reload: reloadMock },
    });
    render(<AppErrorFallback error={new Error('test error')} />);
    const btn = document.querySelector('button')!;
    fireEvent.click(btn);
    expect(reloadMock).toHaveBeenCalled();
  });
});
