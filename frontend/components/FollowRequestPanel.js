'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { getFollowRequests, acceptFollowRequest, declineFollowRequest } from '../services/followers';

export default function FollowRequestsPanel({ onCountChange }) {
  const [requests, setRequests] = useState([]);
  const [loading, setLoading] = useState(true);
  const [acting, setActing] = useState(null); // requestId being acted on

  const load = async () => {
    try {
      const data = await getFollowRequests();
      setRequests(data.requests ?? []);
      onCountChange?.(data.requests?.length ?? 0);
    } catch {
      setRequests([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const handle = async (requestId, action) => {
    setActing(requestId);
    try {
      if (action === 'accept') await acceptFollowRequest(requestId);
      else await declineFollowRequest(requestId);
      setRequests(prev => prev.filter(r => r.id !== requestId));
      onCountChange?.(prev => Math.max(0, prev - 1));
    } catch {
      // ignore
    } finally {
      setActing(null);
    }
  };

  if (loading) return null;
  if (requests.length === 0) return null;

  return (
    <div style={{
      background: 'var(--bg-surface)',
      border: '1px solid var(--border)',
      borderRadius: 'var(--radius-md)',
      padding: '24px 32px',
      marginBottom: 24,
    }}>
      <h2 style={{
        fontSize: 11,
        letterSpacing: '0.1em',
        textTransform: 'uppercase',
        color: 'var(--text-muted)',
        marginBottom: 16,
      }}>
        Follow Requests ({requests.length})
      </h2>

      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        {requests.map(req => (
          <div key={req.id} style={{
            display: 'flex',
            alignItems: 'center',
            gap: 12,
            padding: '10px 0',
            borderBottom: '1px solid var(--border)',
          }}>
            {/* Avatar */}
            <Link href={`/profile/${req.sender.id}`}>
              <div style={{
                width: 36, height: 36, borderRadius: '50%',
                background: 'var(--bg-elevated)',
                border: '1px solid var(--border)',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                fontSize: 13, color: 'var(--accent)', fontWeight: 500,
                flexShrink: 0, overflow: 'hidden', cursor: 'pointer',
              }}>
                {req.sender.avatar_path
                  ? <img src={`/api/proxy${req.sender.avatar_path}`} alt="" style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
                  : `${req.sender.first_name?.[0] ?? ''}${req.sender.last_name?.[0] ?? ''}`.toUpperCase()
                }
              </div>
            </Link>

            <div style={{ flex: 1 }}>
              <Link href={`/profile/${req.sender.id}`} style={{ fontSize: 14, color: 'var(--text-primary)', fontWeight: 500 }}>
                {req.sender.first_name} {req.sender.last_name}
              </Link>
              {req.sender.nickname && (
                <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>@{req.sender.nickname}</div>
              )}
            </div>

            <div style={{ display: 'flex', gap: 8 }}>
              <button
                onClick={() => handle(req.id, 'accept')}
                disabled={acting === req.id}
                style={acceptBtn}
              >
                Accept
              </button>
              <button
                onClick={() => handle(req.id, 'decline')}
                disabled={acting === req.id}
                style={declineBtn}
              >
                Decline
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

const acceptBtn = {
  background: 'var(--accent)',
  color: '#0d0d0d',
  border: 'none',
  borderRadius: 'var(--radius-sm)',
  padding: '6px 14px',
  fontSize: 12,
  fontWeight: 500,
  cursor: 'pointer',
  fontFamily: 'var(--font-body)',
};

const declineBtn = {
  background: 'none',
  color: 'var(--text-secondary)',
  border: '1px solid var(--border)',
  borderRadius: 'var(--radius-sm)',
  padding: '6px 14px',
  fontSize: 12,
  cursor: 'pointer',
  fontFamily: 'var(--font-body)',
};