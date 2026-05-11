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

// GET /posts
export function getFeedPosts({ limit = 20, beforeId } = {}) {
  return apiRequest(`/posts${toQuery({ limit, before_id: beforeId })}`);
}

// POST /posts
export function createPost(data) {
  return apiRequest('/posts', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

// GET /posts/:id
export function getPostById(postId) {
  return apiRequest(`/posts/${postId}`);
}

// DELETE /posts/:id
export function deletePost(postId) {
  return apiRequest(`/posts/${postId}`, { method: 'DELETE' });
}

// POST /posts/:id/image (multipart)
export async function uploadPostImage(postId, file) {
  const form = new FormData();
  form.append('image', file);

  const response = await fetch(`${API_BASE}/posts/${postId}/image`, {
    method: 'POST',
    credentials: 'include',
    body: form,
  });

  if (!response.ok) {
    const err = await response.json().catch(() => ({}));
    throw new Error(err.error || `HTTP ${response.status}`);
  }

  return response.json();
}

// GET /posts/my-followers
export function getMyFollowersForPostVisibility() {
  return apiRequest('/posts/my-followers');
}

// GET /users/:id/posts
export function getUserPosts(userId, { limit = 20, beforeId } = {}) {
  return apiRequest(`/users/${userId}/posts${toQuery({ limit, before_id: beforeId })}`);
}

// GET /groups/:groupID/posts
export function getGroupPosts(groupId, { limit = 20, beforeId } = {}) {
  return apiRequest(`/groups/${groupId}/posts${toQuery({ limit, before_id: beforeId })}`);
}

// POST /groups/:groupID/posts
export function createGroupPost(groupId, data) {
  return apiRequest(`/groups/${groupId}/posts`, {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

// POST /groups/:groupID/posts/:postID/image (multipart)
export async function uploadGroupPostImage(groupId, postId, file) {
  const form = new FormData();
  form.append('image', file);

  const response = await fetch(`${API_BASE}/groups/${groupId}/posts/${postId}/image`, {
    method: 'POST',
    credentials: 'include',
    body: form,
  });

  if (!response.ok) {
    const err = await response.json().catch(() => ({}));
    throw new Error(err.error || `HTTP ${response.status}`);
  }

  return response.json();
}

// GET /posts/:postID/comments
export function getComments(postId) {
  return apiRequest(`/posts/${postId}/comments`);
}

// POST /posts/:postID/comments
export function createComment(postId, data) {
  return apiRequest(`/posts/${postId}/comments`, {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

// DELETE /posts/:postID/comments/:commentID
export function deleteComment(postId, commentId) {
  return apiRequest(`/posts/${postId}/comments/${commentId}`, {
    method: 'DELETE',
  });
}
