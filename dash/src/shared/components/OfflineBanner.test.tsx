import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import * as useNetworkStatusHook from '../hooks/useNetworkStatus';
import { OfflineBanner } from './OfflineBanner';

vi.mock('../hooks/useNetworkStatus');

describe('OfflineBanner', () => {
  it('renders banner when offline', () => {
    vi.mocked(useNetworkStatusHook.useNetworkStatus).mockReturnValue({
      isOnline: false,
    });
    render(<OfflineBanner />);
    expect(screen.getByText(/You are offline/i)).toBeInTheDocument();
  });

  it('renders nothing when online', () => {
    vi.mocked(useNetworkStatusHook.useNetworkStatus).mockReturnValue({
      isOnline: true,
    });
    const { container } = render(<OfflineBanner />);
    expect(container).toBeEmptyDOMElement();
  });
});
