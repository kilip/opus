import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { OfflineBanner } from './OfflineBanner';
import * as useNetworkStatusHook from '../hooks/useNetworkStatus';

vi.mock('../hooks/useNetworkStatus');

describe('OfflineBanner', () => {
  it('renders banner when offline', () => {
    vi.mocked(useNetworkStatusHook.useNetworkStatus).mockReturnValue({ isOnline: false });
    render(<OfflineBanner />);
    expect(screen.getByText(/You are offline/i)).toBeInTheDocument();
  });

  it('renders nothing when online', () => {
    vi.mocked(useNetworkStatusHook.useNetworkStatus).mockReturnValue({ isOnline: true });
    const { container } = render(<OfflineBanner />);
    expect(container).toBeEmptyDOMElement();
  });
});
