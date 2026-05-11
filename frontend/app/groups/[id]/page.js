'use client';

import { useState, useEffect } from 'react';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import AuthGuard from '../../../components/AuthGuard';
import AppShell from '../../../components/AppShell';
import { useAuth } from '../../../context/AuthContext';
import {
  getGroup,
  requestJoin,
  listMembers,
  getGroupPosts,
  listEvents,
  createEvent,
  respondEvent,
  listJoinRequests,
  acceptJoinRequest,
  declineJoinRequest,
  inviteToGroup,
} from '../../../services/groups';
import { createPost, uploadGroupPostImage } from '../../../services/posts';

export default function GroupDetailPage() {
  return (
    <AuthGuard>
      <AppShell>
        <GroupDetail />
      </AppShell>
    </AuthGuard>
  );
}

function GroupDetail() {
  const params = useParams();
  const id = params?.id;
  const { user } = useAuth();
  const [group, setGroup] = useState(null);
  const [tab, setTab] = useState('posts');
  const [posts, setPosts] = useState([]);
  const [members, setMembers] = useState([]);
  const [events, setEvents] = useState([]);
  const [joinRequests, setJoinRequests] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!id) return;
    getGroup(id)
      .then((d) => { setGroup(d.group); })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, [id]);

  useEffect(() => {
    if (!group || !id) return;
    if (tab === 'posts' && group.viewer_status?.is_member) {
      getGroupPosts(id).then((d) => setPosts(d.posts ?? [])).catch(() => {});
    }
    if (tab === 'members') {
      listMembers(id).then((d) => setMembers(d.members ?? [])).catch(() => {});
    }
    if (tab === 'events' && group.viewer_status?.is_member) {
      listEvents(id).then((d) => setEvents(d.events ?? [])).catch(() => {});
    }
    if (tab === 'requests' && group.creator_id === user?.id) {
      listJoinRequests(id).then((d) => setJoinRequests(d.requests ?? [])).catch(() => {});
    }
  }, [tab, group, id, user]);

  const handleJoin = async () => {
    try {
      await requestJoin(id);
      setGroup((g) => ({ ...g, viewer_status: { ...g.viewer_status, has_pending_join_request: true } }));
    } catch {}
  };

  const handleAcceptRequest = async (reqId) => {
    await acceptJoinRequest(reqId);
    setJoinRequests((prev) => prev.filter((r) => r.id !== reqId));
  };

  const handleDeclineRequest = async (reqId) => {
    await declineJoinRequest(reqId);
    setJoinRequests((prev) => prev.filter((r) => r.id !== reqId));
  };

  if (loading) return <Spinner />;
  if (error) return <ErrorCard msg={error} />;
  if (!group) return null;

  const status = group.viewer_status || {};
  const isCreator = group.creator_id === user?.id;
  const isMember = status.is_member;

  const tabs = ['posts', 'members', 'events'];
  if (isCreator) tabs.push('requests');

  return (
    <div style={{ maxWidth: 860, margin: '0 auto', padding: '32px 24px' }}>
      <div style={{ background: 'var(--bg-surface)', border: '1px solid var(--border)', borderRadius: 12, padding: '24px', marginBottom: 24 }}>
        <div style={{ display: 'flex', alignItems: 'flex-start', gap: 20 }}>
          <div style={{
            width: 64, height: 64, borderRadius: 12, background: 'var(--bg-elevated)',
            border: '1px solid var(--border)', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0,
          }}>
            <i className="ti ti-users" style={{ fontSize: 28, color: 'var(--accent)' }} aria-hidden="true" />
          </div>
          <div style={{ flex: 1 }}>
            <h1 style={{ fontFamily: 'var(--font-display)', fontSize: 24, color: 'var(--text-primary)', margin: '0 0 4px' }}>{group.title}</h1>
            <p style={{ fontSize: 13, color: 'var(--text-muted)', margin: '0 0 8px' }}>{group.member_count} members</p>
            {group.description && <p style={{ fontSize: 14, color: 'var(--text-secondary)', lineHeight: 1.6, margin: 0 }}>{group.description}</p>}
          </div>

          <div style={{ display: 'flex', gap: 8 }}>
            {!isMember && !isCreator && (
              <button
                onClick={handleJoin}
                disabled={status.has_pending_join_request}
                style={{ ...primaryBtn, opacity: status.has_pending_join_request ? 0.6 : 1 }}
              >
                {status.has_pending_join_request ? 'Requested' : 'Request to join'}
              </button>
            )}
            {isMember && <span style={{ fontSize: 13, color: 'var(--text-muted)', padding: '8px 0' }}>✓ Member</span>}
            {isCreator && <span style={{ fontSize: 13, color: 'var(--accent)', padding: '8px 0' }}>Creator</span>}
          </div>
        </div>
      </div>

      <div style={{ display: 'flex', gap: 4, marginBottom: 20, borderBottom: '1px solid var(--border)' }}>
        {tabs.map((t) => (
          <button key={t} onClick={() => setTab(t)} style={{
            background: 'none', border: 'none', cursor: 'pointer',
            fontFamily: 'var(--font-body)', fontSize: 14, fontWeight: tab === t ? 600 : 400,
            color: tab === t ? 'var(--text-primary)' : 'var(--text-secondary)',
            padding: '8px 16px', borderBottom: tab === t ? '2px solid var(--accent)' : '2px solid transparent',
            marginBottom: -1, textTransform: 'capitalize',
          }}>{t}</button>
        ))}
      </div>

      {tab === 'posts' && (
        <PostsTab groupId={id} isMember={isMember} posts={posts} setPosts={setPosts} />
      )}
      {tab === 'members' && (
        <MembersTab members={members} groupId={id} isCreator={isCreator} />
      )}
      {tab === 'events' && (
        <EventsTab groupId={id} isMember={isMember} events={events} setEvents={setEvents} />
      )}
      {tab === 'requests' && isCreator && (
        <RequestsTab requests={joinRequests} onAccept={handleAcceptRequest} onDecline={handleDeclineRequest} />
      )}
    </div>
  );
}

function PostsTab({ groupId, isMember, posts, setPosts }) {
  const [body, setBody] = useState('');
  const [posting, setPosting] = useState(false);
  const [imageFile, setImageFile] = useState(null);
  const [imagePreview, setImagePreview] = useState('');
  const [submitError, setSubmitError] = useState('');

  const submit = async (e) => {
    e.preventDefault();
    if (!body.trim() && !imageFile) return;
    if (!body.trim()) {
      setSubmitError('Add a caption/message, then click Send.');
      return;
    }
    setPosting(true);
    setSubmitError('');
    try {
      const data = await createPost({ body: body.trim(), privacy: 'public', group_id: groupId });
      let post = data.post;
      if (imageFile && post?.id) {
        const img = await uploadGroupPostImage(groupId, post.id, imageFile);
        post = { ...post, image_path: img.image_path ?? post.image_path };
      }
      setPosts((prev) => [post, ...prev]);
      setBody('');
      setImageFile(null);
      setImagePreview('');
    } catch (err) {
      setSubmitError(err?.message || 'Failed to send post');
    }
    setPosting(false);
  };

  return (
    <div>
      {isMember && (
        <form onSubmit={submit} style={{ background: 'var(--bg-surface)', border: '1px solid var(--border)', borderRadius: 12, padding: 16, marginBottom: 20 }}>
          <textarea
            value={body}
            onChange={(e) => setBody(e.target.value)}
            placeholder="Write something to the group..."
            rows={3}
            style={{ width: '100%', resize: 'none', background: 'transparent', border: 'none', color: 'var(--text-primary)', fontSize: 15, fontFamily: 'var(--font-body)', outline: 'none' }}
          />
          {imagePreview ? (
            <div style={{ marginTop: 8 }}>
              <img src={imagePreview} alt="" style={{ width: '100%', borderRadius: 8, maxHeight: 320, objectFit: 'cover' }} />
            </div>
          ) : null}
          {!submitError && imageFile ? (
            <p style={{ marginTop: 8, fontSize: 12, color: 'var(--text-muted)' }}>
              Image selected. Add text, then click Send.
            </p>
          ) : null}
          {submitError ? (
            <p style={{ marginTop: 8, fontSize: 12, color: 'var(--danger)' }}>
              {submitError}
            </p>
          ) : null}
          <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 8 }}>
            <label style={{ ...secondaryBtn, marginRight: 8, cursor: 'pointer' }}>
              Image
              <input
                type="file"
                accept=".jpg,.jpeg,.png,.gif"
                style={{ display: 'none' }}
                onChange={(e) => {
                  const f = e.target.files?.[0];
                  if (!f) return;
                  setImageFile(f);
                  setImagePreview(URL.createObjectURL(f));
                  setSubmitError('');
                }}
              />
            </label>
            <button type="submit" disabled={(!body.trim() && !imageFile) || posting} style={{ ...primaryBtn, opacity: ((!body.trim() && !imageFile) || posting) ? 0.5 : 1 }}>Send</button>
          </div>
        </form>
      )}
      {posts.length === 0
        ? <p style={{ color: 'var(--text-muted)', fontSize: 14 }}>{isMember ? 'No posts yet. Be the first!' : 'Join the group to see posts.'}</p>
        : posts.map((p) => <MiniPost key={p.id} post={p} />)
      }
    </div>
  );
}

function MiniPost({ post }) {
  const author = post.author;
  return (
    <div style={{ background: 'var(--bg-surface)', border: '1px solid var(--border)', borderRadius: 12, padding: 16, marginBottom: 12 }}>
      <div style={{ display: 'flex', gap: 10, alignItems: 'center', marginBottom: 10 }}>
        <div style={{ width: 36, height: 36, borderRadius: '50%', background: 'var(--bg-elevated)', border: '1px solid var(--border)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 13, color: 'var(--accent)' }}>
          {author?.first_name?.[0]}{author?.last_name?.[0]}
        </div>
        <div>
          <div style={{ fontSize: 14, fontWeight: 600, color: 'var(--text-primary)' }}>{author?.first_name} {author?.last_name}</div>
          <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>{new Date(post.created_at).toLocaleDateString()}</div>
        </div>
      </div>
      <p style={{ fontSize: 14, color: 'var(--text-primary)', lineHeight: 1.6, margin: 0 }}>{post.body}</p>
      {post.image_path && <img src={`/api/proxy${post.image_path}`} alt="" style={{ width: '100%', borderRadius: 8, marginTop: 10, maxHeight: 400, objectFit: 'cover' }} />}
    </div>
  );
}

function MembersTab({ members, groupId, isCreator }) {
  const [inviteId, setInviteId] = useState('');
  const [inviteMsg, setInviteMsg] = useState('');

  const handleInvite = async (e) => {
    e.preventDefault();
    if (!inviteId.trim()) return;
    try {
      await inviteToGroup(groupId, inviteId.trim());
      setInviteMsg('Invitation sent!');
      setInviteId('');
    } catch (err) {
      setInviteMsg(err.message);
    }
  };

  return (
    <div>
      {isCreator && (
        <form onSubmit={handleInvite} style={{ background: 'var(--bg-surface)', border: '1px solid var(--border)', borderRadius: 12, padding: 16, marginBottom: 20, display: 'flex', gap: 10, alignItems: 'center' }}>
          <input value={inviteId} onChange={(e) => setInviteId(e.target.value)} placeholder="User ID to invite" style={{ ...inputStyle, flex: 1 }} />
          <button type="submit" style={primaryBtn}>Invite</button>
          {inviteMsg && <span style={{ fontSize: 13, color: 'var(--text-muted)' }}>{inviteMsg}</span>}
        </form>
      )}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
        {members.map((m) => (
          <div key={m.user.id} style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '10px 16px', background: 'var(--bg-surface)', border: '1px solid var(--border)', borderRadius: 10, marginBottom: 4 }}>
            <div style={{ width: 36, height: 36, borderRadius: '50%', background: 'var(--bg-elevated)', border: '1px solid var(--border)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 13, color: 'var(--accent)' }}>
              {m.user.avatar_path
                ? <img src={`/api/proxy${m.user.avatar_path}`} alt="" style={{ width: '100%', height: '100%', objectFit: 'cover', borderRadius: '50%' }} />
                : `${m.user.first_name?.[0] ?? ''}${m.user.last_name?.[0] ?? ''}`}
            </div>
            <div style={{ flex: 1 }}>
              <Link href={`/profile/${m.user.id}`} style={{ fontSize: 14, fontWeight: 500, color: 'var(--text-primary)', textDecoration: 'none' }}>
                {m.user.first_name} {m.user.last_name}
              </Link>
              {m.user.nickname && <span style={{ fontSize: 12, color: 'var(--text-muted)', marginLeft: 6 }}>@{m.user.nickname}</span>}
            </div>
            <span style={{ fontSize: 11, color: m.role === 'creator' ? 'var(--accent)' : 'var(--text-muted)', textTransform: 'capitalize' }}>{m.role}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function EventsTab({ groupId, isMember, events, setEvents }) {
  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState({ title: '', description: '', event_time: '' });

  const handleCreate = async (e) => {
    e.preventDefault();
    try {
      const data = await createEvent(groupId, form);
      setEvents((prev) => [...prev, data.event]);
      setCreating(false);
      setForm({ title: '', description: '', event_time: '' });
    } catch {}
  };

  const handleRsvp = async (eventId, response) => {
    try {
      const data = await respondEvent(groupId, eventId, response);
      setEvents((prev) => prev.map((ev) => ev.id === eventId ? data.event : ev));
    } catch {}
  };

  return (
    <div>
      {isMember && (
        <button onClick={() => setCreating(true)} style={{ ...primaryBtn, marginBottom: 20 }}>
          <i className="ti ti-calendar-plus" aria-hidden="true" /> Create event
        </button>
      )}

      {creating && (
        <>
          <div onClick={() => setCreating(false)} style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.5)', zIndex: 200 }} />
          <div style={{ position: 'fixed', top: '50%', left: '50%', transform: 'translate(-50%,-50%)', background: 'var(--bg-surface)', border: '1px solid var(--border)', borderRadius: 16, padding: 32, width: 440, zIndex: 201 }}>
            <h2 style={{ fontFamily: 'var(--font-display)', fontSize: 20, color: 'var(--text-primary)', marginBottom: 20 }}>Create event</h2>
            <form onSubmit={handleCreate} style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                <label style={labelStyle}>Title</label>
                <input value={form.title} onChange={(e) => setForm((f) => ({ ...f, title: e.target.value }))} placeholder="Event title" style={inputStyle} required />
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                <label style={labelStyle}>Description</label>
                <textarea value={form.description} onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))} rows={3} style={{ ...inputStyle, resize: 'vertical' }} />
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                <label style={labelStyle}>Date & time</label>
                <input type="datetime-local" value={form.event_time ? new Date(form.event_time).toISOString().slice(0, 16) : ''} onChange={(e) => setForm((f) => ({ ...f, event_time: new Date(e.target.value).toISOString() }))} style={inputStyle} required />
              </div>
              <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
                <button type="button" onClick={() => setCreating(false)} style={secondaryBtn}>Cancel</button>
                <button type="submit" style={primaryBtn}>Create</button>
              </div>
            </form>
          </div>
        </>
      )}

      {events.length === 0
        ? <p style={{ color: 'var(--text-muted)', fontSize: 14 }}>{isMember ? 'No events yet.' : 'Join to see events.'}</p>
        : events.map((ev) => (
          <div key={ev.id} style={{ background: 'var(--bg-surface)', border: '1px solid var(--border)', borderRadius: 12, padding: 20, marginBottom: 12 }}>
            <div style={{ display: 'flex', alignItems: 'flex-start', gap: 16 }}>
              <div style={{ textAlign: 'center', background: 'var(--bg-elevated)', border: '1px solid var(--border)', borderRadius: 10, padding: '8px 14px', flexShrink: 0 }}>
                <div style={{ fontSize: 11, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                  {new Date(ev.event_time).toLocaleDateString('en-US', { month: 'short' })}
                </div>
                <div style={{ fontSize: 22, fontWeight: 600, color: 'var(--text-primary)', fontFamily: 'var(--font-display)' }}>
                  {new Date(ev.event_time).getDate()}
                </div>
              </div>
              <div style={{ flex: 1 }}>
                <h3 style={{ fontSize: 16, fontWeight: 600, color: 'var(--text-primary)', margin: '0 0 4px' }}>{ev.title}</h3>
                {ev.description && <p style={{ fontSize: 13, color: 'var(--text-secondary)', margin: '0 0 8px' }}>{ev.description}</p>}
                <div style={{ fontSize: 12, color: 'var(--text-muted)', marginBottom: 10 }}>
                  <i className="ti ti-clock" aria-hidden="true" /> {new Date(ev.event_time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                  &nbsp;·&nbsp; ✅ {ev.going_count} going &nbsp;·&nbsp; ❌ {ev.not_going_count} not going
                </div>
                {isMember && (
                  <div style={{ display: 'flex', gap: 8 }}>
                    <button
                      onClick={() => handleRsvp(ev.id, 'going')}
                      style={{ ...(ev.viewer_response === 'going' ? primaryBtn : secondaryBtn), fontSize: 12, padding: '6px 14px' }}
                    >
                      ✅ Going
                    </button>
                    <button
                      onClick={() => handleRsvp(ev.id, 'not_going')}
                      style={{ ...(ev.viewer_response === 'not_going' ? primaryBtn : secondaryBtn), fontSize: 12, padding: '6px 14px' }}
                    >
                      ❌ Can't go
                    </button>
                  </div>
                )}
              </div>
            </div>
          </div>
        ))
      }
    </div>
  );
}

function RequestsTab({ requests, onAccept, onDecline }) {
  return (
    <div>
      {requests.length === 0
        ? <p style={{ color: 'var(--text-muted)', fontSize: 14 }}>No pending join requests.</p>
        : requests.map((req) => (
          <div key={req.id} style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '12px 16px', background: 'var(--bg-surface)', border: '1px solid var(--border)', borderRadius: 10, marginBottom: 8 }}>
            <div style={{ flex: 1, fontSize: 14, color: 'var(--text-primary)' }}>
              {req.user.first_name} {req.user.last_name}
              {req.user.nickname && <span style={{ fontSize: 12, color: 'var(--text-muted)', marginLeft: 6 }}>@{req.user.nickname}</span>}
            </div>
            <button onClick={() => onAccept(req.id)} style={{ ...primaryBtn, fontSize: 12, padding: '6px 14px' }}>Accept</button>
            <button onClick={() => onDecline(req.id)} style={{ ...secondaryBtn, fontSize: 12, padding: '6px 14px' }}>Decline</button>
          </div>
        ))
      }
    </div>
  );
}

function Spinner() {
  return (
    <div style={{ display: 'flex', justifyContent: 'center', padding: 60 }}>
      <div style={{ width: 24, height: 24, borderRadius: '50%', border: '2px solid var(--border)', borderTopColor: 'var(--accent)', animation: 'spin 0.8s linear infinite' }} />
      <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
    </div>
  );
}

function ErrorCard({ msg }) {
  return (
    <div style={{ maxWidth: 600, margin: '48px auto', padding: '0 24px' }}>
      <div style={{ background: 'rgba(192,87,74,0.1)', border: '1px solid rgba(192,87,74,0.3)', borderRadius: 12, padding: 24, color: '#e87060', fontSize: 14 }}>{msg}</div>
    </div>
  );
}

const primaryBtn = {
  background: 'var(--accent)', color: '#0d0d0d', border: 'none',
  borderRadius: 8, padding: '8px 16px', fontSize: 13, fontWeight: 500,
  cursor: 'pointer', fontFamily: 'var(--font-body)', display: 'inline-flex', alignItems: 'center', gap: 6,
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
