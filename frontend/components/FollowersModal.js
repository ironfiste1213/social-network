'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { getFollowers, getFollowing } from '../services/followers';

/**
 * FollowersModal
 * Props:
 *   userId    – whose followers/following to show
 *   mode      – 'followers' | 'following'
 *   onClose   – callback to close modal
 */
export default function FollowersModal({ userId, mode, onClose }) {
  const [list, setList] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fn = mode === 'followers' ? getFollowers : getFollowing;
    fn(userId)
      .then(data => setList(data[mode] ?? []))
      .catch(() => setList([]))
      .finally(() => setLoading(false));
  }, [userId, mode]);

  return (
    <>
      {/* Backdrop */}
      <div
        onClick={onClose}
        style={{
          position: 'fixed', inset: 0,
          background: 'rgba(0,0,0,0.6)',
          zIndex: 200,
        }}
      />

      {/* Modal */}
      <div style={{
        position: 'fixed',
        top: '50%', left: '50%',
        transform: 'translate(-50%, -50%)',
        background: 'var(--bg-surface)',
        border: '1px solid var(--border)',
        borderRadius: 'var(--radius-md)',
        width: 380,
        maxHeight: '70vh',
        display: 'flex',
        flexDirection: 'column',
        zIndex: 201,
        overflow: 'hidden',
      }}>
        {/* Header */}
        <div style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '16px 20px',
          borderBottom: '1px solid var(--border)',
        }}>
          <h3 style={{
            fontFamily: 'var(--font-display)',
            fontSize: 18,
            color: 'var(--text-primary)',
          }}>
            {mode === 'followers' ? 'Followers' : 'Following'}
          </h3>
          <button
            onClick={onClose}
            style={{
              background: 'none', border: 'none',
              color: 'var(--text-muted)', fontSize: 20,
              cursor: 'pointer', lineHeight: 1,
            }}
          >
            ×
          </button>
        </div>

        {/* List */}
        <div style={{ overflowY: 'auto', flex: 1 }}>
          {loading ? (
            <div style={{ padding: 24, textAlign: 'center', color: 'var(--text-muted)', fontSize: 13 }}>
              Loading…
            </div>
          ) : list.length === 0 ? (
            <div style={{ padding: 24, textAlign: 'center', color: 'var(--text-muted)', fontSize: 13 }}>
              {mode === 'followers' ? 'No followers yet.' : 'Not following anyone yet.'}
            </div>
          ) : (
            list.map(user => (
              <Link
                key={user.id}
                href={`/profile/${user.id}`}
                onClick={onClose}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 12,
                  padding: '12px 20px',
                  borderBottom: '1px solid var(--border)',
                  transition: 'background var(--transition)',
                }}
                onMouseEnter={e => e.currentTarget.style.background = 'var(--bg-elevated)'}
                onMouseLeave={e => e.currentTarget.style.background = 'transparent'}
              >
                <div style={{
                  width: 36, height: 36, borderRadius: '50%',
                  background: 'var(--bg-elevated)',
                  border: '1px solid var(--border)',
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  fontSize: 13, color: 'var(--accent)', fontWeight: 500,
                  flexShrink: 0, overflow: 'hidden',
                }}>
                  {user.avatar_path
                    ? <img src={`/api/proxy${user.avatar_path}`} alt="" style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
                    : `${user.first_name?.[0] ?? ''}${user.last_name?.[0] ?? ''}`.toUpperCase()
                  }
                </div>
                <div>
                  <div style={{ fontSize: 14, color: 'var(--text-primary)', fontWeight: 500 }}>
                    {user.first_name} {user.last_name}
                  </div>
                  {user.nickname && (
                    <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>@{user.nickname}</div>
                  )}
                </div>
              </Link>
            ))
          )}
        </div>
      </div>
    </>
  );
}