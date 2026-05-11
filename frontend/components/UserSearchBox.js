'use client';

import Link from 'next/link';
import { useEffect, useRef, useState } from 'react';
import { searchUsers } from '../services/users';

const MIN_QUERY_LENGTH = 2;

export default function UserSearchBox({ autoFocus = false, onSelect }) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);
  const [error, setError] = useState('');
  const rootRef = useRef(null);
  const inputRef = useRef(null);

  useEffect(() => {
    if (autoFocus) inputRef.current?.focus();
  }, [autoFocus]);

  useEffect(() => {
    function handleDocumentClick(event) {
      if (rootRef.current && !rootRef.current.contains(event.target)) {
        setOpen(false);
      }
    }
    document.addEventListener('mousedown', handleDocumentClick);
    return () => document.removeEventListener('mousedown', handleDocumentClick);
  }, []);

  useEffect(() => {
    const trimmedQuery = query.trim();
    if (trimmedQuery.length < MIN_QUERY_LENGTH) {
      setResults([]); setLoading(false); setError('');
      return undefined;
    }
    setLoading(true); setError('');
    const timeoutId = window.setTimeout(async () => {
      try {
        const data = await searchUsers(trimmedQuery);
        setResults(data.users ?? []);
        setOpen(true);
      } catch (err) {
        setResults([]); setError(err.message || 'Search failed'); setOpen(true);
      } finally {
        setLoading(false);
      }
    }, 250);
    return () => window.clearTimeout(timeoutId);
  }, [query]);

  const trimmedQuery = query.trim();
  const showPanel = open && trimmedQuery.length >= MIN_QUERY_LENGTH;

  return (
    <div ref={rootRef} style={{ position: 'relative', width: '100%' }}>
      <div style={{ position: 'relative' }}>
        <i className="ti ti-search" style={{ position: 'absolute', left: 12, top: '50%', transform: 'translateY(-50%)', fontSize: 16, color: 'var(--text-muted)', pointerEvents: 'none' }} aria-hidden="true" />
        <input
          ref={inputRef}
          type="search"
          value={query}
          placeholder="Search by nickname…"
          aria-label="Search users"
          onFocus={() => { if (trimmedQuery.length >= MIN_QUERY_LENGTH) setOpen(true); }}
          onChange={(e) => { setQuery(e.target.value); if (!open) setOpen(true); }}
          style={{
            width: '100%',
            height: 44,
            borderRadius: 10,
            border: '1px solid var(--border)',
            background: 'var(--bg-elevated)',
            color: 'var(--text-primary)',
            padding: '0 14px 0 38px',
            fontSize: 14,
            outline: 'none',
            fontFamily: 'var(--font-body)',
          }}
        />
      </div>

      {showPanel && (
        <div style={{
          position: 'absolute',
          top: 'calc(100% + 6px)',
          left: 0, right: 0,
          background: 'var(--bg-elevated)',
          border: '1px solid var(--border)',
          borderRadius: 12,
          overflow: 'hidden',
          zIndex: 200,
          boxShadow: '0 8px 24px rgba(0,0,0,0.2)',
        }}>
          {loading && <PanelMessage>Searching…</PanelMessage>}
          {!loading && error && <PanelMessage tone="error">{error}</PanelMessage>}
          {!loading && !error && results.length === 0 && <PanelMessage>No users found.</PanelMessage>}
          {!loading && !error && results.map((user) => (
            <Link
              key={user.id}
              href={`/profile/${user.id}`}
              onClick={() => { setOpen(false); setQuery(''); onSelect?.(); }}
              style={{
                display: 'flex', alignItems: 'center', gap: 12,
                padding: '10px 14px',
                color: 'inherit', textDecoration: 'none',
                borderBottom: '1px solid var(--border)',
                transition: 'background var(--transition)',
              }}
              onMouseEnter={e => e.currentTarget.style.background = 'var(--bg-surface)'}
              onMouseLeave={e => e.currentTarget.style.background = 'transparent'}
            >
              <Avatar user={user} />
              <div style={{ minWidth: 0, flex: 1 }}>
                <div style={{ fontSize: 14, color: 'var(--text-primary)', fontWeight: 500, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {user.first_name} {user.last_name}
                </div>
                <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>
                  @{user.nickname}
                  <span style={{
                    marginLeft: 8, padding: '1px 6px', borderRadius: 6,
                    background: user.profile_visibility === 'private' ? 'rgba(90,158,111,0.15)' : 'rgba(201,185,154,0.15)',
                    color: user.profile_visibility === 'private' ? 'var(--success)' : 'var(--accent)',
                    fontSize: 11,
                  }}>{user.profile_visibility}</span>
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}

function Avatar({ user }) {
  const src = user.avatar_path ? `/api/proxy${user.avatar_path}` : null;
  const initials = `${user.first_name?.[0] ?? ''}${user.last_name?.[0] ?? ''}`.toUpperCase();
  return (
    <div style={{
      width: 36, height: 36, borderRadius: '50%',
      background: 'var(--bg-surface)', border: '1px solid var(--border)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      overflow: 'hidden', flexShrink: 0, color: 'var(--accent)', fontSize: 12, fontWeight: 600,
    }}>
      {src ? <img src={src} alt="" style={{ width: '100%', height: '100%', objectFit: 'cover' }} /> : initials}
    </div>
  );
}

function PanelMessage({ children, tone = 'default' }) {
  return (
    <div style={{ padding: '12px 14px', fontSize: 13, color: tone === 'error' ? 'var(--danger)' : 'var(--text-muted)' }}>
      {children}
    </div>
  );
}
