// frontend/services/auth.js
// Authentication API layer for session-based backend

const API_BASE = '/api/proxy';  // Proxy to backend to avoid CORS

async function apiRequest(url, options = {}) {
  const config = {
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
    },
    ...options
  };

  const response = await fetch(`${API_BASE}${url}`, config);
  
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP ${response.status}`);
  }
  
  return response.json();
}

// register(data) - POST /auth/register
export async function register(data) {
  return apiRequest('/auth/register', {
    method: 'POST',
    body: JSON.stringify(data)
  });
}

// login(data) - POST /auth/login
export async function login(data) {
  return apiRequest('/auth/login', {
    method: 'POST',
    body: JSON.stringify(data)
  });
}

// logout() - POST /auth/logout
export async function logout() {
  return apiRequest('/auth/logout', {
    method: 'POST'
  });
}

// getCurrentUser() - GET /auth/me
export async function getCurrentUser() {
  return apiRequest('/auth/me');
}

