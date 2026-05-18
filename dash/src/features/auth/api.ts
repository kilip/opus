import { queryOptions } from '@tanstack/react-query';
import { api } from '@/shared/lib/api';
import type { AuthUser, LoginCredentials } from './types';

export type { AuthUser, LoginCredentials };

/**
 * Authentication query keys and functions.
 * Consistent with ADR-011.
 */
export const authQueries = {
  me: () =>
    queryOptions({
      queryKey: ['auth', 'me'],
      queryFn: async () => {
        // Mock implementation for development
        // In production, this calls the real endpoint
        if (import.meta.env.DEV && !import.meta.env.VITE_API_URL) {
          await new Promise((resolve) => setTimeout(resolve, 500));
          const isAuthed = localStorage.getItem('opus_mock_authed') === 'true';
          if (!isAuthed) throw new Error('Unauthorized');

          return {
            id: 'usr_01HZ9XYZ',
            email: 'admin@opus.ai',
            name: 'Pak Bos',
            role: 'admin',
            workspace_id: 'ws_01ABCDEF',
          } as AuthUser;
        }

        return api.get<AuthUser>('/auth/me');
      },
      retry: false,
      staleTime: 5 * 60 * 1000, // 5 minutes
    }),
};

/**
 * Perform login with credentials.
 */
export async function login(credentials: LoginCredentials): Promise<AuthUser> {
  // Mock implementation for development
  if (import.meta.env.DEV && !import.meta.env.VITE_API_URL) {
    await new Promise((resolve) => setTimeout(resolve, 1000));

    // Simple mock check
    if (credentials.email === 'admin@opus.ai') {
      localStorage.setItem('opus_mock_authed', 'true');
      return {
        id: 'usr_01HZ9XYZ',
        email: 'admin@opus.ai',
        name: 'Pak Bos',
        role: 'admin',
        workspace_id: 'ws_01ABCDEF',
      } as AuthUser;
    }

    throw new Error('Invalid credentials');
  }

  return api.post<AuthUser>('/auth/login', credentials);
}

/**
 * Perform logout and clear session.
 */
export async function logout(): Promise<void> {
  // Mock implementation for development
  if (import.meta.env.DEV && !import.meta.env.VITE_API_URL) {
    localStorage.removeItem('opus_mock_authed');
    return;
  }

  return api.post<void>('/auth/logout');
}

/**
 * Trigger OAuth2 flow for the given provider.
 */
export function initiateOAuth(provider: 'google' | 'github'): void {
  const baseUrl = import.meta.env.VITE_API_URL ?? '';
  window.location.href = `${baseUrl}/auth/oauth/${provider}`;
}
