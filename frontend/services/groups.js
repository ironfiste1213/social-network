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

function toQuery(params = {}) {
  const query = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      query.set(key, String(value));
    }
  });
  const qs = query.toString();
  return qs ? `?${qs}` : '';
}

// GET /groups
export const listGroups = ({ limit = 20, beforeId } = {}) =>
  apiRequest(`/groups${toQuery({ limit, before_id: beforeId })}`);

// GET /groups/:id
export const getGroup = (id) =>
  apiRequest(`/groups/${id}`);

// POST /groups
export const createGroup = (data) =>
  apiRequest('/groups', { method: 'POST', body: JSON.stringify(data) });

// POST /groups/:groupId/join
export const requestJoin = (groupId) =>
  apiRequest(`/groups/${groupId}/join`, { method: 'POST' });

// POST /groups/:groupId/invite
export const inviteToGroup = (groupId, inviteeId) =>
  apiRequest(`/groups/${groupId}/invite`, {
    method: 'POST',
    body: JSON.stringify({ invitee_id: inviteeId }),
  });

// GET /groups/:groupId/requests
export const listJoinRequests = (groupId) =>
  apiRequest(`/groups/${groupId}/requests`);

// POST /groups/requests/:requestId/accept
export const acceptJoinRequest = (requestId) =>
  apiRequest(`/groups/requests/${requestId}/accept`, { method: 'POST' });

// POST /groups/requests/:requestId/decline
export const declineJoinRequest = (requestId) =>
  apiRequest(`/groups/requests/${requestId}/decline`, { method: 'POST' });

// GET /groups/invitations
export const listInvitations = () =>
  apiRequest('/groups/invitations');

// POST /groups/invitations/:id/accept
export const acceptInvitation = (id) =>
  apiRequest(`/groups/invitations/${id}/accept`, { method: 'POST' });

// POST /groups/invitations/:id/decline
export const declineInvitation = (id) =>
  apiRequest(`/groups/invitations/${id}/decline`, { method: 'POST' });

// GET /groups/:groupId/members
export const listMembers = (groupId) =>
  apiRequest(`/groups/${groupId}/members`);

// GET /groups/:groupId/posts
export const getGroupPosts = (groupId, { limit = 20, beforeId } = {}) =>
  apiRequest(`/groups/${groupId}/posts${toQuery({ limit, before_id: beforeId })}`);

// GET /groups/:groupId/events
export const listEvents = (groupId) =>
  apiRequest(`/groups/${groupId}/events`);

// POST /groups/:groupId/events
export const createEvent = (groupId, data) =>
  apiRequest(`/groups/${groupId}/events`, { method: 'POST', body: JSON.stringify(data) });

// POST /groups/:groupId/events/:eventId/respond
export const respondEvent = (groupId, eventId, response) =>
  apiRequest(`/groups/${groupId}/events/${eventId}/respond`, {
    method: 'POST',
    body: JSON.stringify({ response }),
  });

