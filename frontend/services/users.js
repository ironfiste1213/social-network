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

// GET /users/me
export const getMyProfile = () => apiRequest('/users/me');

// PATCH /users/me
export const updateProfile = (data) =>
  apiRequest('/users/me', {
    method: 'PATCH',
    body: JSON.stringify(data),
  });

// POST /users/me/avatar  (multipart)
export async function uploadAvatar(file) {
  const form = new FormData();
  form.append('avatar', file);
  const response = await fetch(`${API_BASE}/users/me/avatar`, {
    method: 'POST',
    credentials: 'include',
    body: form,
    // No Content-Type header — browser sets multipart boundary automatically
  });
  if (!response.ok) {
    const err = await response.json().catch(() => ({}));
    throw new Error(err.error || `HTTP ${response.status}`);
  }
  return response.json();
}

// GET /users/:id
export const getUserById = (id) => apiRequest(`/users/${id}`);

// GET /users/search?q=:query&limit=:limit
export const searchUsers = (query, limit = 8) =>
  apiRequest(`/users/search?q=${encodeURIComponent(query)}&limit=${limit}`);
