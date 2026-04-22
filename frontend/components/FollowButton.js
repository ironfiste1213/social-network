'use client';

import { useState, useEffect } from 'react';
import { followUser, unfollowUser, getFollowStatus } from '../services/followers';

/**
 * FollowButton
 * Props:
 *   targetId   – the user to follow/unfollow
 *   onUpdate   – optional callback after state changes
 */
export default function FollowButton({ targetId, onUpdate }) {
  const [status, setStatus] = useState(null); // null = loading
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!targetId) return;
    getFollowStatus(targetId)
      .then(setStatus)
      .catch(() => setStatus({ is_following: false, has_pending_request: false }));
  }, [targetId]);

  const handleClick = async () => {
    if (!status || loading) return;
    setLoading(true);
    try {
      if (status.is_following || status.has_pending_request) {
        await unfollowUser(targetId);
        setStatus({ is_following: false, has_pending_request: false });
      } else {
        const result = await followUser(targetId);
        // If result has a request_id it means a follow request was sent (private profile)
        // Otherwise it was a direct follow (public profile)
        setStatus({
          is_following: !result?.request_id,
          has_pending_request: !!result?.request_id,
        });
      }
      onUpdate?.();
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  };

  if (status === null) {
    return (
      <button style={btnBase} disabled>
        <span style={dot} />
      </button>
    );
  }

  const label = loading
    ? '…'
    : status.is_following
    ? 'Following'
    : status.has_pending_request
    ? 'Requested'
    : 'Follow';

  const active = status.is_following || status.has_pending_request;

  return (
    <button
      onClick={handleClick}
      disabled={loading}
      style={{
        ...btnBase,
        background: active ? 'var(--bg-elevated)' : 'var(--accent)',
        color: active ? 'var(--text-secondary)' : '#0d0d0d',
        border: active ? '1px solid var(--border)' : '1px solid transparent',
        opacity: loading ? 0.6 : 1,
      }}
    >
      {label}
    </button>
  );
}

const btnBase = {
  padding: '7px 18px',
  borderRadius: 'var(--radius-sm)',
  fontSize: 13,
  fontWeight: 500,
  fontFamily: 'var(--font-body)',
  cursor: 'pointer',
  transition: 'opacity var(--transition), background var(--transition)',
  minWidth: 90,
};

const dot = {
  display: 'inline-block',
  width: 8,
  height: 8,
  borderRadius: '50%',
  background: 'var(--text-muted)',
};