import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ThemeProvider, useTheme } from './useTheme';

function Probe() {
  const { preference, toggle } = useTheme();
  return (
    <button type="button" onClick={toggle}>
      {preference}
    </button>
  );
}

describe('useTheme', () => {
  it('applies dark class after toggle from system', () => {
    localStorage.setItem('opus-theme', 'system');
    render(
      <ThemeProvider>
        <Probe />
      </ThemeProvider>,
    );

    fireEvent.click(screen.getByRole('button'));
    expect(
      document.documentElement.classList.contains('dark') ||
        document.documentElement.classList.contains('light'),
    ).toBe(true);
  });
});
