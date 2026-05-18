/**
 * User representation from the server.
 * Consistent with ADR-011 and server models.
 */
export interface AuthUser {
  id: string;
  email: string;
  name: string;
  role: 'admin' | 'user';
  workspace_id: string;
  avatar_url?: string;
}

/**
 * Login credentials for local authentication.
 */
export interface LoginCredentials {
  email: string;
  password?: string;
}
