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

export const getConversations = () =>
  apiRequest('/chat/conversations');

export const getPrivateHistory = (receiverId, beforeId = '', limit = 50) => {
  const params = new URLSearchParams({ receiver_id: receiverId, limit: String(limit) });
  if (beforeId) params.set('before_id', beforeId);
  return apiRequest(`/chat/messages?${params.toString()}`);
};

export const getGroupHistory = (groupId, beforeId = '', limit = 50) => {
  const params = new URLSearchParams({ group_id: groupId, limit: String(limit) });
  if (beforeId) params.set('before_id', beforeId);
  return apiRequest(`/chat/messages?${params.toString()}`);
};

// WebSocket helper for /ws backend endpoint
export function createChatSocket() {
  const defaultBase = `${window.location.protocol === 'https:' ? 'wss' : 'ws'}://localhost:8080`;
  const base = process.env.NEXT_PUBLIC_WS_URL || defaultBase;
  return new WebSocket(`${base}/ws`);
}

// ---- outbound event helpers (must match backend InboundEvent) ----
export function sendPrivate(socket, toUserId, body) {
  socket.send(JSON.stringify({ type: 'send_private', to: toUserId, body }));
}

export function sendGroup(socket, groupId, body) {
  socket.send(JSON.stringify({ type: 'send_group', to: groupId, body }));
}

export function pingChat(socket) {
  socket.send(JSON.stringify({ type: 'ping' }));
}

