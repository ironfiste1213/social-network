'use client';

import { useState, useEffect } from 'react';
import { useParams } from 'next/navigation';
import AuthGuard from '../../../components/AuthGuard';
import AppShell from '../../../components/AppShell';
import FollowButton from '../../../components/FollowButton';
import FollowersModal from '../../../components/FollowersModal';
import { getUserById } from '../../../services/users';
import { getFollowers, getFollowing } from '../../../services/followers';
import { useAuth } from '../../../context/AuthContext';

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
  const [profile, setProfile]   = useState(null);
  const [loading, setLoading]   = useState(true);
  const [error, setError]       = useState('');
  const [counts, setCounts]     = useState({ followers: 0, following: 0 });
  const [modal, setModal]       = useState(null);

  useEffect(() => {
    if (!id) return;
    if (me && me.id === id) {
      window.location.replace('/profile');
      return;
    }
    getUserById(id)
      .then(data => {
        setProfile(data.user);
        // Load follow counts for public profiles
        if (!data.user?.is_private) {
          Promise.all([getFollowers(id), getFollowing(id)])
            .then(([frs, fing]) => setCounts({
              followers: frs.followers?.length ?? 0,
              following: fing.following?.length ?? 0,
            }))
            .catch(() => {});
        }
      })
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
          background: 'var(--bg-surface)', border: '1px solid var(--border)',
          borderRadius: 'var(--radius-md)', padding: '48px 32px', textAlign: 'center',
        }}>
          <Avatar name={profile} size={72} />
          <h2 style={{ fontFamily: 'var(--font-display)', fontSize: 22, color: 'var(--text-primary)', margin: '16px 0 8px' }}>
            {profile.first_name} {profile.last_name}
          </h2>
          <p style={{ fontSize: 13, color: 'var(--text-muted)', marginBottom: 24 }}>This account is private.</p>
          <FollowButton targetId={id} />
        </div>
      </div>
    );
  }

  const joined = profile.created_at
    ? new Date(profile.created_at).toLocaleDateString('en-US', { year: 'numeric', month: 'long' })
    : '—';

  return (
    <div style={{ maxWidth: 680, margin: '0 auto', padding: '48px 24px' }}>
      <div style={{
        background: 'var(--bg-surface)', border: '1px solid var(--border)',
        borderRadius: 'var(--radius-md)', padding: '32px', marginBottom: 24,
      }}>
        <div style={{ display: 'flex', alignItems: 'flex-start', gap: 24, marginBottom: 20 }}>
          <Avatar name={profile} size={80} avatarPath={profile.avatar_path} />

          <div style={{ flex: 1 }}>
            <h1 style={{
              fontFamily: 'var(--font-display)', fontSize: 26,
              letterSpacing: '-0.5px', color: 'var(--text-primary)', marginBottom: 4,
            }}>
              {profile.first_name} {profile.last_name}
            </h1>
            {profile.nickname && (
              <p style={{ fontSize: 13, color: 'var(--accent)', marginBottom: 4 }}>@{profile.nickname}</p>
            )}
            <p style={{ fontSize: 12, color: 'var(--text-muted)' }}>Joined {joined}</p>
          </div>

          <FollowButton targetId={id} />
        </div>

        {profile.about_me && (
          <p style={{ fontSize: 14, color: 'var(--text-secondary)', lineHeight: 1.6, marginBottom: 20 }}>
            {profile.about_me}
          </p>
        )}

        <div style={{ display: 'flex', gap: 24 }}>
          <Stat label="Posts" value="0" />
          <Stat
            label="Followers"
            value={counts.followers}
            onClick={() => setModal('followers')}
            clickable
          />
          <Stat
            label="Following"
            value={counts.following}
            onClick={() => setModal('following')}
            clickable
          />
        </div>
      </div>

      {/* Posts placeholder */}
      <div style={{
        background: 'var(--bg-surface)', border: '1px solid var(--border)',
        borderRadius: 'var(--radius-md)', padding: '32px', textAlign: 'center',
      }}>
        <p style={{ fontSize: 14, color: 'var(--text-muted)' }}>No posts yet.</p>
      </div>

      {modal && (
        <FollowersModal userId={id} mode={modal} onClose={() => setModal(null)} />
      )}
    </div>
  );
}

function Avatar({ name, size = 40, avatarPath }) {
  const initials = `${name?.first_name?.[0] ?? ''}${name?.last_name?.[0] ?? ''}`.toUpperCase();
  const src = avatarPath ? `/api/proxy${avatarPath}` : null;
  return (
    <div style={{
      width: size, height: size, borderRadius: '50%',
      background: 'var(--bg-elevated)', border: '2px solid var(--border)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      fontSize: size * 0.3, color: 'var(--accent)', fontWeight: 500,
      flexShrink: 0, overflow: 'hidden', margin: size === 72 ? '0 auto' : undefined,
    }}>
      {src
        ? <img src={src} alt="avatar" style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
        : initials
      }
    </div>
  );
}

function Stat({ label, value, onClick, clickable }) {
  return (
    <div onClick={onClick} style={{ cursor: clickable ? 'pointer' : 'default' }}>
      <div style={{
        fontSize: 20, fontFamily: 'var(--font-display)',
        color: clickable ? 'var(--accent)' : 'var(--text-primary)',
      }}>
        {value}
      </div>
      <div style={{ fontSize: 11, color: 'var(--text-muted)', letterSpacing: '0.06em', textTransform: 'uppercase' }}>
        {label}
      </div>
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
        background: 'rgba(192,87,74,0.1)', border: '1px solid rgba(192,87,74,0.3)',
        borderRadius: 'var(--radius-md)', padding: '24px', fontSize: 14, color: '#e87060',
      }}>
        {message}
      </div>
    </div>
  );
}