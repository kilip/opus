// dash/src/shared/lib/api-client.ts
const BASE_URL = import.meta.env?.VITE_API_URL ?? 'http://localhost:8080';

export const apiClient = {
  get: async <T>(path: string): Promise<T> => {
    const res = await fetch(`${BASE_URL}${path}`);
    if (!res.ok) throw new Error(res.statusText);
    return res.json() as Promise<T>;
  },
  post: async <T>(path: string, body: unknown): Promise<T> => {
    const res = await fetch(`${BASE_URL}${path}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    });
    if (!res.ok) throw new Error(res.statusText);
    return res.json() as Promise<T>;
  },
};
