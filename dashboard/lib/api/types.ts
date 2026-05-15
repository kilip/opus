export interface ApiResponse<T> {
  success: boolean;
  data: T | null;
  error: ApiError | null;
}

export interface ApiError {
  code: string;
  message: string;
}

export interface User {
  id: string;
  email: string;
  name: string;
  avatarUrl: string;
  provider: string;
  createdAt: string;
  updatedAt: string;
}

export interface AuthTokens {
  accessToken: string;
}

export interface StreamEvent {
  type: string;
  data: string;
  timestamp: string;
}
