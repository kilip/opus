// dash/src/shared/utils/api.test.ts
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { api } from './api';

describe('api', () => {
  beforeEach(() => {
    globalThis.fetch = vi.fn();
  });

  it('performs GET request', async () => {
    const mockResponse = { data: 'test' };
    vi.mocked(globalThis.fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => mockResponse,
    } as Response);

    const result = await api.get('/test');
    expect(result).toEqual(mockResponse);
    expect(globalThis.fetch).toHaveBeenCalledWith('http://localhost:8080/test');
  });

  it('performs POST request', async () => {
    const mockResponse = { success: true };
    vi.mocked(globalThis.fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => mockResponse,
    } as Response);

    const result = await api.post('/test', { foo: 'bar' });
    expect(result).toEqual(mockResponse);
    expect(globalThis.fetch).toHaveBeenCalledWith('http://localhost:8080/test', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ foo: 'bar' }),
    });
  });
});
