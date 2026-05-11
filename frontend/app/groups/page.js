'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import AuthGuard from '../../components/AuthGuard';
import AppShell from '../../components/AppShell';
import { useAuth } from '../../context/AuthContext';
import {
  listGroups, createGroup, requestJoin,
  listInvitations, acceptInvitation, declineInvitation,
} from '../../services/groups';

export default function GroupsPage() {
  return (
    <AuthGuard>
      <AppShell>
        <GroupsContent />
      </AppShell>
    </AuthGuard>
  );
}

function GroupsContent() {
  const { user } = useAuth();
  const [groups, setGroups] = useState([]);
  const [invitations, setInvitations] = useState([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState({ title: '', description: '' });
  const [formError, setFormError] = useState('');
  const [tab, setTab] = useState('all'); // 'all' | 'mine'

  useEffect(() => {
    Promise.all([
      listGroups({ limit: 30 }),
      listInvitations(),
    ]).then(([gData, iData]) => {
      setGroups(gData.groups ?? []);
      setInvitations(iData.invitations ?? []);
    }).catch(() => {}).finally(() => setLoading(false));
  }, []);

  const handleCreate = async (e) => {
    e.preventDefault();
    if (!form.title.trim()) { setFormError('Title is required'); return; }
    try {
      const data = await createGroup(form);
      setGroups(prev => [data.group, ...prev]);
      setForm({ title: '', description: '' });
      setCreating(false);
    } catch (err) {
      setFormError(err.message);
    }
  };

  const handleJoin = async (groupId) => {
    try {
      await requestJoin(groupId);
      setGroups(prev => prev.map(g => g.id === groupId
        ? { ...g, viewer_status: { ...g.viewer_status, has_pending_join_request: true } }
        : g
      ));
    } catch {}
  };

  const handleAcceptInvite = async (inviteId, groupId) => {
    await acceptInvitation(inviteId);
    setInvitations(prev => prev.filter(i => i.id !== inviteId));
    setGroups(prev => prev.map(g => g.id === groupId
      ? { ...g, viewer_status: { ...g.viewer_status, is_member: true } }
      : g
    ));
  };

  const handleDeclineInvite = async (inviteId) => {
    await declineInvitation(inviteId);
    setInvitations(prev => prev.filter(i => i.id !== inviteId));
  };

  const displayed = tab === 'mine'
    ? groups.filter(g => g.viewer_status?.is_member)
    : groups;

  return (
    <div style={{ maxWidth: 900, margin: '0 auto', padding: '32px 24px' }}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 24 }}>
        <h1 style={{ fontFamily: 'var(--font-display)', fontSize: 28, color: 'var(--text-primary)', margin: 0 }}>Groups</h1>
        <button onClick={() => setCreating(true)} style={primaryBtn}>
          <i className="ti ti-plus" aria-hidden="true" /> Create group
        </button>
      </div>

      {/* Invitations */}
      {invitations.length > 0 && (
        <div style={{ background: 'var(--bg-surface)', border: '1px solid var(--border)', borderRadius: 12, padding: '16px 20px', marginBottom: 24 }}>
          <h2 style={{ fontSize: 13, letterSpacing: '0.08em', textTransform: 'uppercase', color: 'var(--text-muted)', marginBottom: 12, margin: '0 0 12px' }}>
            Group invitations ({invitations.length})
          </h2>
          {invitations.map(inv => (
            <div key={inv.id} style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '8px 0', borderBottom: '1px solid var(--border)' }}>
              <div style={{ flex: 1 }}>
                <span style={{ fontSize: 14, color: 'var(--text-primary)', fontWeight: 500 }}>{inv.group?.title}</span>
                <span style={{ fontSize: 12, color: 'var(--text-muted)', marginLeft: 8 }}>
                  invited by {inv.inviter?.first_name} {inv.inviter?.last_name}
                </span>
              </div>
              <button onClick={() => handleAcceptInvite(inv.id, inv.group_id)} style={primaryBtn}>Accept</button>
              <button onClick={() => handleDeclineInvite(inv.id)} style={secondaryBtn}>Decline</button>
            </div>
          ))}
        </div>
      )}

      {/* Create form modal */}
      {creating && (
        <>
          <div onClick={() => setCreating(false)} style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.5)', zIndex: 200 }} />
          <div style={{
            position: 'fixed', top: '50%', left: '50%', transform: 'translate(-50%,-50%)',
            background: 'var(--bg-surface)', border: '1px solid var(--border)',
            borderRadius: 16, padding: 32, width: 440, zIndex: 201,
          }}>
            <h2 style={{ fontFamily: 'var(--font-display)', fontSize: 22, color: 'var(--text-primary)', marginBottom: 20 }}>Create group</h2>
            <form onSubmit={handleCreate} style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                <label style={labelStyle}>Group name</label>
                <input value={form.title} onChange={e => setForm(f => ({ ...f, title: e.target.value }))} placeholder="e.g. Photography Club" style={inputStyle} />
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                <label style={labelStyle}>Description</label>
                <textarea value={form.description} onChange={e => setForm(f => ({ ...f, description: e.target.value }))} placeholder="What is this group about?" rows={3} style={{ ...inputStyle, resize: 'vertical' }} />
              </div>
              {formError && <p style={{ fontSize: 13, color: 'var(--danger)', margin: 0 }}>{formError}</p>}
              <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end', marginTop: 4 }}>
                <button type="button" onClick={() => setCreating(false)} style={secondaryBtn}>Cancel</button>
                <button type="submit" style={primaryBtn}>Create</button>
              </div>
            </form>
          </div>
        </>
      )}

      {/* Tabs */}
      <div style={{ display: 'flex', gap: 4, marginBottom: 20, borderBottom: '1px solid var(--border)', paddingBottom: 0 }}>
        {['all', 'mine'].map(t => (
          <button key={t} onClick={() => setTab(t)} style={{
            background: 'none', border: 'none', cursor: 'pointer',
            fontFamily: 'var(--font-body)', fontSize: 14, fontWeight: tab === t ? 600 : 400,
            color: tab === t ? 'var(--text-primary)' : 'var(--text-secondary)',
            padding: '8px 16px',
            borderBottom: tab === t ? '2px solid var(--accent)' : '2px solid transparent',
            marginBottom: -1,
          }}>
            {t === 'all' ? 'Discover' : 'My groups'}
          </button>
        ))}
      </div>

      {loading ? <Spinner /> : (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))', gap: 16 }}>
          {displayed.map(g => (
            <GroupCard key={g.id} group={g} currentUserId={user?.id} onJoin={() => handleJoin(g.id)} />
          ))}
          {displayed.length === 0 && (
            <p style={{ color: 'var(--text-muted)', fontSize: 14, gridColumn: '1/-1' }}>
              {tab === 'mine' ? "You haven't joined any groups yet." : 'No groups yet. Create one!'}
            </p>
          )}
        </div>
      )}
    </div>
  );
}

function GroupCard({ group, currentUserId, onJoin }) {
  const status = group.viewer_status || {};
  const isCreator = group.creator_id === currentUserId;

  const joinLabel = status.is_member
    ? 'Member'
    : status.has_pending_join_request
    ? 'Requested'
    : 'Join';

  return (
    <div style={{
      background: 'var(--bg-surface)', border: '1px solid var(--border)',
      borderRadius: 12, padding: '20px', display: 'flex', flexDirection: 'column', gap: 10,
    }}>
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 12 }}>
        <div style={{
          width: 44, height: 44, borderRadius: 10,
          background: 'var(--bg-elevated)', border: '1px solid var(--border)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          fontSize: 20, flexShrink: 0,
        }}>
          <i className="ti ti-users" style={{ fontSize: 22, color: 'var(--accent)' }} aria-hidden="true" />
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <Link href={`/groups/${group.id}`} style={{ fontSize: 15, fontWeight: 600, color: 'var(--text-primary)', textDecoration: 'none', display: 'block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {group.title}
          </Link>
          <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>
            {group.member_count} member{group.member_count !== 1 ? 's' : ''}
          </div>
        </div>
      </div>

      {group.description && (
        <p style={{ fontSize: 13, color: 'var(--text-secondary)', lineHeight: 1.5, margin: 0,
          overflow: 'hidden', display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical' }}>
          {group.description}
        </p>
      )}

      <div style={{ display: 'flex', gap: 8, marginTop: 4 }}>
        <Link href={`/groups/${group.id}`} style={{ ...secondaryBtn, textDecoration: 'none', textAlign: 'center', flex: 1 }}>View</Link>
        {!isCreator && !status.is_member && (
          <button
            onClick={onJoin}
            disabled={status.has_pending_join_request}
            style={{ ...primaryBtn, flex: 1, opacity: status.has_pending_join_request ? 0.6 : 1 }}
          >
            {joinLabel}
          </button>
        )}
      </div>
    </div>
  );
}

function Spinner() {
  return (
    <div style={{ display: 'flex', justifyContent: 'center', padding: 40 }}>
      <div style={{ width: 24, height: 24, borderRadius: '50%', border: '2px solid var(--border)', borderTopColor: 'var(--accent)', animation: 'spin 0.8s linear infinite' }} />
      <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
    </div>
  );
}

const primaryBtn = {
  background: 'var(--accent)', color: '#0d0d0d', border: 'none',
  borderRadius: 8, padding: '8px 16px', fontSize: 13, fontWeight: 500,
  cursor: 'pointer', fontFamily: 'var(--font-body)', display: 'inline-flex',
  alignItems: 'center', gap: 6,
};
const secondaryBtn = {
  background: 'none', color: 'var(--text-secondary)', border: '1px solid var(--border)',
  borderRadius: 8, padding: '8px 16px', fontSize: 13, cursor: 'pointer', fontFamily: 'var(--font-body)',
};
const inputStyle = {
  background: 'var(--bg-input)', border: '1px solid var(--border)',
  borderRadius: 8, padding: '10px 14px', color: 'var(--text-primary)',
  fontSize: 14, fontFamily: 'var(--font-body)', outline: 'none', width: '100%',
};
const labelStyle = {
  fontSize: 11, letterSpacing: '0.08em', textTransform: 'uppercase', color: 'var(--text-muted)', fontWeight: 500,
};
