const API_BASE = '/api/proxy';

async function apiRequest(url, options = {}) {
  const config = {
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    ...options,
  };
  const response = await fetch(`${API_BASE}${url}`, config);
  if (!response.ok) {
    const err = await response.json().catch(() => ({}));
    throw new Error(err.error || `HTTP ${response.status}`);
  }
  return response.json();
}

export const register     = (data) => apiRequest('/auth/register', { method: 'POST', body: JSON.stringify(data) });
export const login        = (data) => apiRequest('/auth/login',    { method: 'POST', body: JSON.stringify(data) });
export const logout       = ()     => apiRequest('/auth/logout',   { method: 'POST' });
export const getCurrentUser = ()   => apiRequest('/auth/me');


