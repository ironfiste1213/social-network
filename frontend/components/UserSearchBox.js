'use client';

import Link from 'next/link';
import { useEffect, useRef, useState } from 'react';
import { searchUsers } from '../services/users';

const MIN_QUERY_LENGTH = 2;

export default function UserSearchBox() {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);
  const [error, setError] = useState('');
  const rootRef = useRef(null);

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
      setResults([]);
      setLoading(false);
      setError('');
      return undefined;
    }

    setLoading(true);
    setError('');

    const timeoutId = window.setTimeout(async () => {
      try {
        const data = await searchUsers(trimmedQuery);
        setResults(data.users ?? []);
        setOpen(true);
      } catch (err) {
        setResults([]);
        setError(err.message || 'Search failed');
        setOpen(true);
      } finally {
        setLoading(false);
      }
    }, 250);

    return () => window.clearTimeout(timeoutId);
  }, [query]);

  const trimmedQuery = query.trim();
  const showPanel = open && trimmedQuery.length >= MIN_QUERY_LENGTH;

  return (
    <div ref={rootRef} style={{ position: 'relative', width: 'min(320px, 40vw)' }}>
      <input
        type="search"
        value={query}
        placeholder="Search by username"
        aria-label="Search users by username"
        onFocus={() => {
          if (trimmedQuery.length >= MIN_QUERY_LENGTH) {
            setOpen(true);
          }
        }}
        onChange={(event) => {
          setQuery(event.target.value);
          if (!open) {
            setOpen(true);
          }
        }}
        style={{
          width: '100%',
          height: 38,
          borderRadius: 999,
          border: '1px solid var(--border)',
          background: 'var(--bg-surface)',
          color: 'var(--text-primary)',
          padding: '0 14px',
          fontSize: 13,
          outline: 'none',
        }}
      />

      {showPanel && (
        <div style={{
          position: 'absolute',
          top: 'calc(100% + 10px)',
          left: 0,
          right: 0,
          background: 'var(--bg-surface)',
          border: '1px solid var(--border)',
          borderRadius: 'var(--radius-md)',
          boxShadow: '0 18px 40px rgba(0, 0, 0, 0.12)',
          overflow: 'hidden',
          zIndex: 200,
        }}>
          {loading && (
            <PanelMessage>Searching...</PanelMessage>
          )}

          {!loading && error && (
            <PanelMessage tone="error">{error}</PanelMessage>
          )}

          {!loading && !error && results.length === 0 && (
            <PanelMessage>No users found.</PanelMessage>
          )}

          {!loading && !error && results.map((user) => (
            <Link
              key={user.id}
              href={`/profile/${user.id}`}
              onClick={() => {
                setOpen(false);
                setQuery('');
              }}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 12,
                padding: '12px 14px',
                color: 'inherit',
                textDecoration: 'none',
                borderTop: '1px solid var(--border)',
              }}
            >
              <Avatar user={user} />

              <div style={{ minWidth: 0, flex: 1 }}>
                <div style={{ fontSize: 13, color: 'var(--text-primary)', fontWeight: 500, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                  {user.first_name} {user.last_name}
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 12, color: 'var(--text-muted)' }}>
                  <span>@{user.nickname}</span>
                  <span style={{
                    padding: '2px 7px',
                    borderRadius: 999,
                    background: user.profile_visibility === 'private' ? 'rgba(31, 96, 75, 0.12)' : 'rgba(194, 122, 51, 0.12)',
                    color: user.profile_visibility === 'private' ? '#1f604b' : '#a25d20',
                    textTransform: 'capitalize',
                  }}>
                    {user.profile_visibility}
                  </span>
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
      width: 36,
      height: 36,
      borderRadius: '50%',
      background: 'var(--bg-elevated)',
      border: '1px solid var(--border)',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      overflow: 'hidden',
      flexShrink: 0,
      color: 'var(--accent)',
      fontSize: 12,
      fontWeight: 600,
    }}>
      {src
        ? <img src={src} alt="" style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
        : initials
      }
    </div>
  );
}

function PanelMessage({ children, tone = 'default' }) {
  return (
    <div style={{
      padding: '14px',
      fontSize: 12,
      color: tone === 'error' ? '#c0574a' : 'var(--text-muted)',
    }}>
      {children}
    </div>
  );
}
