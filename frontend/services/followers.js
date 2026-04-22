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

// Follow a user (POST /follow/{id})
export const followUser = (id) =>
  apiRequest(`/follow/${id}`, { method: 'POST' });

// Unfollow a user (DELETE /follow/{id})
export const unfollowUser = (id) =>
  apiRequest(`/follow/${id}`, { method: 'DELETE' });

// Get incoming pending follow requests
export const getFollowRequests = () =>
  apiRequest('/follow/requests');

// Accept a follow request
export const acceptFollowRequest = (requestId) =>
  apiRequest(`/follow/requests/${requestId}/accept`, { method: 'POST' });

// Decline a follow request
export const declineFollowRequest = (requestId) =>
  apiRequest(`/follow/requests/${requestId}/decline`, { method: 'POST' });

// Get followers list for a user
export const getFollowers = (userId) =>
  apiRequest(`/users/${userId}/followers`);

// Get following list for a user
export const getFollowing = (userId) =>
  apiRequest(`/users/${userId}/following`);

// Get follow status from current user → target user
// Returns { is_following, has_pending_request, request_id }
export const getFollowStatus = (userId) =>
  apiRequest(`/users/${userId}/follow-status`);