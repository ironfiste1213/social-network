'use client';

import { useState, useEffect } from 'react';
import { useParams } from 'next/navigation';
import AuthGuard from '../../../components/AuthGuard';
import AppShell from '../../../components/AppShell';
import { getUserById } from '../../../services/users';
import { useAuth } from '../../../context/AuthContext';
import Link from 'next/link';

export default function UserProfilePage() {
  return (
    <AuthGuard>
      <AppShell>
        <UserProfileContent />
      </AppShell>
    </AuthGuard>
  );
}

function UserProfileContent() {
  const { id } = useParams();
  const { user: me } = useAuth();
  const [profile, setProfile] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError]     = useState('');

  useEffect(() => {
    if (!id) return;
    // If viewing own profile, redirect
    if (me && me.id === id) {
      window.location.replace('/profile');
      return;
    }
    getUserById(id)
      .then(data => setProfile(data.user))
      .catch(e => setError(e.message))
      .finally(() => setLoading(false));
  }, [id, me]);

  if (loading) return <Spinner />;
  if (error)   return <ErrorCard message={error} />;
  if (!profile) return null;

  // Private profile — limited view
  if (profile.is_private) {
    return (
      <div style={{ maxWidth: 680, margin: '0 auto', padding: '48px 24px' }}>
        <div style={{
          background: 'var(--bg-surface)',
          border: '1px solid var(--border)',
          borderRadius: 'var(--radius-md)',
          padding: '48px 32px',
          textAlign: 'center',
        }}>
          <div style={{
            width: 72, height: 72, borderRadius: '50%',
            background: 'var(--bg-elevated)',
            border: '2px solid var(--border)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            fontSize: 22, color: 'var(--accent)', fontWeight: 500,
            margin: '0 auto 16px',
          }}>
            {profile.first_name?.[0]?.toUpperCase() ?? '?'}
          </div>
          <h2 style={{ fontFamily: 'var(--font-display)', fontSize: 22, color: 'var(--text-primary)', marginBottom: 8 }}>
            {profile.first_name} {profile.last_name}
          </h2>
          <p style={{ fontSize: 13, color: 'var(--text-muted)', marginBottom: 24 }}>
            This account is private.
          </p>
          <p style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
            Follow this user to see their profile and posts.
          </p>
        </div>
      </div>
    );
  }

  const initials = `${profile.first_name?.[0] ?? ''}${profile.last_name?.[0] ?? ''}`.toUpperCase();
  const avatarSrc = profile.avatar_path ? `/api/proxy${profile.avatar_path}` : null;
  const joined = profile.created_at
    ? new Date(profile.created_at).toLocaleDateString('en-US', { year: 'numeric', month: 'long' })
    : '—';

  return (
    <div style={{ maxWidth: 680, margin: '0 auto', padding: '48px 24px' }}>
      <div style={{
        background: 'var(--bg-surface)',
        border: '1px solid var(--border)',
        borderRadius: 'var(--radius-md)',
        padding: '32px',
        marginBottom: 24,
      }}>
        <div style={{ display: 'flex', alignItems: 'flex-start', gap: 24, marginBottom: 20 }}>
          {/* Avatar */}
          <div style={{
            width: 80, height: 80, borderRadius: '50%',
            background: 'var(--bg-elevated)',
            border: '2px solid var(--border)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            fontSize: 24, color: 'var(--accent)', fontWeight: 500,
            flexShrink: 0, overflow: 'hidden',
          }}>
            {avatarSrc
              ? <img src={avatarSrc} alt="avatar" style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
              : initials
            }
          </div>

          <div style={{ flex: 1 }}>
            <h1 style={{
              fontFamily: 'var(--font-display)',
              fontSize: 26, letterSpacing: '-0.5px',
              color: 'var(--text-primary)', marginBottom: 4,
            }}>
              {profile.first_name} {profile.last_name}
            </h1>
            {profile.nickname && (
              <p style={{ fontSize: 13, color: 'var(--accent)', marginBottom: 4 }}>@{profile.nickname}</p>
            )}
            <p style={{ fontSize: 12, color: 'var(--text-muted)' }}>Joined {joined}</p>
          </div>
        </div>

        {profile.about_me && (
          <p style={{ fontSize: 14, color: 'var(--text-secondary)', lineHeight: 1.6, marginBottom: 20 }}>
            {profile.about_me}
          </p>
        )}

        <div style={{ display: 'flex', gap: 24 }}>
          <Stat label="Posts"     value="0" />
          <Stat label="Followers" value="0" />
          <Stat label="Following" value="0" />
        </div>
      </div>

      {/* Posts placeholder */}
      <div style={{
        background: 'var(--bg-surface)',
        border: '1px solid var(--border)',
        borderRadius: 'var(--radius-md)',
        padding: '32px',
        textAlign: 'center',
      }}>
        <p style={{ fontSize: 14, color: 'var(--text-muted)' }}>
          No posts yet.
        </p>
      </div>
    </div>
  );
}

function Stat({ label, value }) {
  return (
    <div>
      <div style={{ fontSize: 20, fontFamily: 'var(--font-display)', color: 'var(--text-primary)' }}>{value}</div>
      <div style={{ fontSize: 11, color: 'var(--text-muted)', letterSpacing: '0.06em', textTransform: 'uppercase' }}>{label}</div>
    </div>
  );
}

function Spinner() {
  return (
    <div style={{ minHeight: '50vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <div style={{
        width: 28, height: 28, borderRadius: '50%',
        border: '1.5px solid var(--border)', borderTopColor: 'var(--accent)',
        animation: 'spin 0.8s linear infinite',
      }} />
      <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
    </div>
  );
}

function ErrorCard({ message }) {
  return (
    <div style={{ maxWidth: 680, margin: '48px auto', padding: '0 24px' }}>
      <div style={{
        background: 'rgba(192,87,74,0.1)',
        border: '1px solid rgba(192,87,74,0.3)',
        borderRadius: 'var(--radius-md)',
        padding: '24px', fontSize: 14, color: '#e87060',
      }}>
        {message}
      </div>
    </div>
  );
}