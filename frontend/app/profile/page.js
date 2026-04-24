'use client';

import { useState, useEffect, useRef } from 'react';
import AuthGuard from '../../components/AuthGuard';
import AppShell from '../../components/AppShell';
import FollowRequestsPanel from '../../components/FollowRequestsPanel';
import FollowersModal from '../../components/FollowersModal';
import { useAuth } from '../../context/AuthContext';
import { updateProfile, uploadAvatar } from '../../services/users';
import { getFollowers, getFollowing } from '../../services/followers';

export default function ProfilePage() {
  return (
    <AuthGuard>
      <AppShell>
        <ProfileContent />
      </AppShell>
    </AuthGuard>
  );
}

function ProfileContent() {
  const { user, refreshUser } = useAuth();
  const [editing, setEditing]         = useState(false);
  const [saving, setSaving]           = useState(false);
  const [error, setError]             = useState('');
  const [success, setSuccess]         = useState('');
  const [avatarPreview, setAvatarPreview] = useState(null);
  const [avatarFile, setAvatarFile]       = useState(null);
  const [modal, setModal]             = useState(null); // 'followers' | 'following' | null
  const [counts, setCounts]           = useState({ followers: 0, following: 0 });
  const fileRef = useRef(null);

  const [form, setForm] = useState({
    nickname: '', about_me: '', profile_visibility: 'public',
  });

  useEffect(() => {
    if (user) {
      setForm({
        nickname: user.nickname || '',
        about_me: user.about_me || '',
        profile_visibility: user.profile_visibility || 'public',
      });
      // Load counts
      Promise.all([getFollowers(user.id), getFollowing(user.id)])
        .then(([frs, fing]) => setCounts({
          followers: frs.followers?.length ?? 0,
          following: fing.following?.length ?? 0,
        }))
        .catch(() => {});
    }
  }, [user]);

  const set = (k) => (e) => setForm(f => ({ ...f, [k]: e.target.value }));

  const handleAvatarChange = (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setAvatarFile(file);
    setAvatarPreview(URL.createObjectURL(file));
  };

  const handleSave = async () => {
    setSaving(true); setError(''); setSuccess('');
    try {
      if (avatarFile) await uploadAvatar(avatarFile);
      await updateProfile({
        nickname: form.nickname || null,
        about_me: form.about_me || null,
        profile_visibility: form.profile_visibility,
      });
      await refreshUser();
      setSuccess('Profile updated.');
      setEditing(false);
      setAvatarFile(null);
      setAvatarPreview(null);
    } catch (e) {
      setError(e.message);
    } finally {
      setSaving(false);
    }
  };

  if (!user) return null;

  const initials = `${user.first_name?.[0] ?? ''}${user.last_name?.[0] ?? ''}`.toUpperCase();
  const avatarSrc = avatarPreview || (user.avatar_path ? `/api/proxy${user.avatar_path}` : null);
  const dob = user.date_of_birth
    ? new Date(user.date_of_birth).toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric' })
    : '—';
  const joined = user.created_at
    ? new Date(user.created_at).toLocaleDateString('en-US', { year: 'numeric', month: 'long' })
    : '—';

  return (
    <div style={{ maxWidth: 680, margin: '0 auto', padding: '48px 24px' }}>

      {/* Follow requests — only shown if there are pending ones */}
      <FollowRequestsPanel />

      {/* Header card */}
      <div style={{
        background: 'var(--bg-surface)',
        border: '1px solid var(--border)',
        borderRadius: 'var(--radius-md)',
        padding: '32px',
        marginBottom: 24,
        position: 'relative',
      }}>
        <div style={{ display: 'flex', alignItems: 'flex-start', gap: 24, marginBottom: 24 }}>
          {/* Avatar */}
          <div
            onClick={() => editing && fileRef.current?.click()}
            style={{
              width: 80, height: 80, borderRadius: '50%',
              background: 'var(--bg-elevated)',
              border: `2px solid ${editing ? 'var(--accent)' : 'var(--border)'}`,
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              fontSize: 24, color: 'var(--accent)', fontWeight: 500,
              flexShrink: 0, cursor: editing ? 'pointer' : 'default',
              overflow: 'hidden', transition: 'border-color var(--transition)',
              position: 'relative',
            }}
          >
            {avatarSrc
              ? <img src={avatarSrc} alt="avatar" style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
              : initials
            }
            {editing && (
              <div style={{
                position: 'absolute', inset: 0,
                background: 'rgba(0,0,0,0.45)',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                fontSize: 11, color: '#fff', letterSpacing: '0.05em',
              }}>CHANGE</div>
            )}
          </div>
          <input ref={fileRef} type="file" accept=".jpg,.jpeg,.png,.gif" style={{ display: 'none' }} onChange={handleAvatarChange} />

          <div style={{ flex: 1 }}>
            <h1 style={{
              fontFamily: 'var(--font-display)',
              fontSize: 26, letterSpacing: '-0.5px',
              color: 'var(--text-primary)', marginBottom: 4,
            }}>
              {user.first_name} {user.last_name}
            </h1>
            {user.nickname && (
              <p style={{ fontSize: 13, color: 'var(--accent)', marginBottom: 4 }}>@{user.nickname}</p>
            )}
            <p style={{ fontSize: 12, color: 'var(--text-muted)' }}>Joined {joined}</p>
          </div>

          <div style={{ display: 'flex', gap: 8 }}>
            {editing ? (
              <>
                <button onClick={() => { setEditing(false); setAvatarPreview(null); setAvatarFile(null); setError(''); }} style={btnSecondary}>Cancel</button>
                <button onClick={handleSave} disabled={saving} style={btnPrimary}>{saving ? 'Saving…' : 'Save'}</button>
              </>
            ) : (
              <button onClick={() => setEditing(true)} style={btnSecondary}>Edit profile</button>
            )}
          </div>
        </div>

        {error   && <div style={errorBox}>{error}</div>}
        {success && <div style={successBox}>{success}</div>}

        {editing ? (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
            <Field label="Nickname">
              <input value={form.nickname} onChange={set('nickname')} placeholder="How friends call you" style={inputStyle} />
            </Field>
            <Field label="About me">
              <textarea value={form.about_me} onChange={set('about_me')} placeholder="A short bio…" rows={4} style={{ ...inputStyle, resize: 'vertical' }} />
            </Field>
            <Field label="Profile visibility">
              <div style={{ display: 'flex', gap: 12 }}>
                {['public', 'private'].map(v => (
                  <label key={v} style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer', fontSize: 14, color: 'var(--text-primary)' }}>
                    <input type="radio" name="visibility" value={v} checked={form.profile_visibility === v} onChange={set('profile_visibility')} style={{ accentColor: 'var(--accent)' }} />
                    {v.charAt(0).toUpperCase() + v.slice(1)}
                  </label>
                ))}
              </div>
            </Field>
          </div>
        ) : (
          <>
            {user.about_me && (
              <p style={{ fontSize: 14, color: 'var(--text-secondary)', lineHeight: 1.6, marginBottom: 16 }}>
                {user.about_me}
              </p>
            )}
            <div style={{ display: 'flex', gap: 24, flexWrap: 'wrap' }}>
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
          </>
        )}
      </div>

      {/* Account info */}
      <div style={{
        background: 'var(--bg-surface)',
        border: '1px solid var(--border)',
        borderRadius: 'var(--radius-md)',
        padding: '24px 32px',
      }}>
        <h2 style={{ fontSize: 11, letterSpacing: '0.1em', textTransform: 'uppercase', color: 'var(--text-muted)', marginBottom: 16 }}>
          Account info
        </h2>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px 32px' }}>
          <InfoRow label="Email"        value={user.email} />
          <InfoRow label="Date of birth" value={dob} />
          <InfoRow label="Visibility"   value={user.profile_visibility} />
          <InfoRow label="Member since" value={joined} />
        </div>
      </div>

      {/* Followers/Following modal */}
      {modal && (
        <FollowersModal
          userId={user.id}
          mode={modal}
          onClose={() => setModal(null)}
        />
      )}
    </div>
  );
}

function Field({ label, children }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
      <label style={{ fontSize: 11, letterSpacing: '0.08em', textTransform: 'uppercase', color: 'var(--text-muted)', fontWeight: 500 }}>
        {label}
      </label>
      {children}
    </div>
  );
}

function Stat({ label, value, onClick, clickable }) {
  return (
    <div
      onClick={onClick}
      style={{ cursor: clickable ? 'pointer' : 'default' }}
    >
      <div style={{
        fontSize: 20,
        fontFamily: 'var(--font-display)',
        color: clickable ? 'var(--accent)' : 'var(--text-primary)',
        transition: 'opacity var(--transition)',
      }}>
        {value}
      </div>
      <div style={{ fontSize: 11, color: 'var(--text-muted)', letterSpacing: '0.06em', textTransform: 'uppercase' }}>{label}</div>
    </div>
  );
}

function InfoRow({ label, value }) {
  return (
    <div>
      <div style={{ fontSize: 11, color: 'var(--text-muted)', letterSpacing: '0.06em', textTransform: 'uppercase', marginBottom: 2 }}>{label}</div>
      <div style={{ fontSize: 14, color: 'var(--text-secondary)' }}>{value}</div>
    </div>
  );
}

const inputStyle = {
  background: 'var(--bg-input)', border: '1px solid var(--border)',
  borderRadius: 'var(--radius-sm)', padding: '10px 14px',
  color: 'var(--text-primary)', fontSize: 14, fontFamily: 'var(--font-body)',
  outline: 'none', width: '100%', transition: 'border-color var(--transition)',
};
const btnPrimary = {
  background: 'var(--accent)', color: '#0d0d0d', border: 'none',
  borderRadius: 'var(--radius-sm)', padding: '8px 18px',
  fontSize: 13, fontWeight: 500, cursor: 'pointer', fontFamily: 'var(--font-body)',
};
const btnSecondary = {
  background: 'none', color: 'var(--text-secondary)', border: '1px solid var(--border)',
  borderRadius: 'var(--radius-sm)', padding: '8px 18px',
  fontSize: 13, cursor: 'pointer', fontFamily: 'var(--font-body)',
};
const errorBox = {
  background: 'rgba(192,87,74,0.1)', border: '1px solid rgba(192,87,74,0.3)',
  borderRadius: 'var(--radius-sm)', padding: '10px 14px',
  fontSize: 13, color: '#e87060', marginBottom: 16,
};
const successBox = {
  background: 'rgba(90,158,111,0.1)', border: '1px solid rgba(90,158,111,0.3)',
  borderRadius: 'var(--radius-sm)', padding: '10px 14px',
  fontSize: 13, color: '#5a9e6f', marginBottom: 16,
};
