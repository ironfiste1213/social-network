'use client';

import { useState, useEffect, useRef } from 'react';
import Link from 'next/link';
import AuthGuard from '../../components/AuthGuard';
import AppShell from '../../components/AppShell';
import { useAuth } from '../../context/AuthContext';
import { getFeedPosts, createPost, deletePost, getMyFollowersForPostVisibility, uploadPostImage, getComments, createComment } from '../../services/posts';

export default function FeedPage() {
  return (
    <AuthGuard>
      <AppShell>
        <FeedContent />
      </AppShell>
    </AuthGuard>
  );
}

function FeedContent() {
  const { user } = useAuth();
  const [posts, setPosts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [beforeId, setBeforeId] = useState('');
  const [hasMore, setHasMore] = useState(true);

  const load = async (reset = false) => {
    const cursor = reset ? '' : beforeId;
    try {
      const data = await getFeedPosts({ limit: 10, beforeId: cursor || undefined });
      const newPosts = data.posts ?? [];
      setPosts((prev) => (reset ? newPosts : [...prev, ...newPosts]));
      setBeforeId(newPosts.length ? newPosts[newPosts.length - 1].id : cursor);
      setHasMore(newPosts.length === 10);
    } catch {
      setHasMore(false);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(true); }, []);

  const onPostCreated = (post) => {
    setPosts((prev) => [post, ...prev]);
  };

  const onPostDeleted = (id) => {
    setPosts((prev) => prev.filter((p) => p.id !== id));
  };

  return (
    <div style={{
      maxWidth: 630,
      margin: '0 auto',
      padding: '32px 16px',
    }}>
      {user && <CreatePost user={user} onCreated={onPostCreated} />}

      {loading ? (
        <Spinner />
      ) : posts.length === 0 ? (
        <div style={{ textAlign: 'center', color: 'var(--text-muted)', fontSize: 14, padding: 48 }}>
          No posts yet. Follow some people to see their posts here!
        </div>
      ) : (
        <>
          {posts.map((post) => (
            <PostCard key={post.id} post={post} currentUser={user} onDeleted={onPostDeleted} />
          ))}
          {hasMore && (
            <button onClick={() => load()} style={loadMoreBtn}>
              Load more
            </button>
          )}
        </>
      )}
    </div>
  );
}

/* ─── Create Post ─── */
function CreatePost({ user, onCreated }) {
  const [body, setBody] = useState('');
  const [privacy, setPrivacy] = useState('public');
  const [viewerIDs, setViewerIDs] = useState([]);
  const [followers, setFollowers] = useState([]);
  const [imageFile, setImageFile] = useState(null);
  const [imagePreview, setImagePreview] = useState(null);
  const [expanded, setExpanded] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const fileRef = useRef(null);

  useEffect(() => {
    if (privacy === 'selected_followers') {
      getMyFollowersForPostVisibility().then((d) => setFollowers(d.followers ?? [])).catch(() => {});
    }
  }, [privacy]);

  const handleImage = (e) => {
    const f = e.target.files?.[0];
    if (!f) return;
    setImageFile(f);
    setImagePreview(URL.createObjectURL(f));
  };

  const submit = async () => {
    if (!body.trim()) return;
    setLoading(true); setError('');
    try {
      const created = await createPost({ body: body.trim(), privacy, viewer_ids: privacy === 'selected_followers' ? viewerIDs : [] });
      let post = created.post;
      if (imageFile && post?.id) {
        const imageRes = await uploadPostImage(post.id, imageFile);
        post = { ...post, image_path: imageRes.image_path ?? post.image_path };
      }
      onCreated(post);
      setBody(''); setImageFile(null); setImagePreview(null); setPrivacy('public'); setViewerIDs([]); setExpanded(false);
    } catch (e) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  };

  const initials = `${user.first_name?.[0] ?? ''}${user.last_name?.[0] ?? ''}`.toUpperCase();
  const avatarSrc = user.avatar_path ? `/api/proxy${user.avatar_path}` : null;

  return (
    <div style={{
      background: 'var(--bg-surface)',
      border: '1px solid var(--border)',
      borderRadius: 12,
      padding: '16px',
      marginBottom: 24,
    }}>
      <div style={{ display: 'flex', gap: 12, alignItems: 'flex-start' }}>
        <AvatarCircle src={avatarSrc} initials={initials} size={40} />
        <div style={{ flex: 1 }}>
          <textarea
            value={body}
            onChange={(e) => { setBody(e.target.value); if (!expanded) setExpanded(true); }}
            onFocus={() => setExpanded(true)}
            placeholder="What's on your mind?"
            rows={expanded ? 3 : 1}
            style={{
              width: '100%', resize: 'none',
              background: 'transparent', border: 'none',
              color: 'var(--text-primary)', fontSize: 15,
              fontFamily: 'var(--font-body)', outline: 'none',
              lineHeight: 1.5,
            }}
          />
          {imagePreview && (
            <div style={{ position: 'relative', marginTop: 8 }}>
              <img src={imagePreview} alt="" style={{ maxWidth: '100%', maxHeight: 300, borderRadius: 8, objectFit: 'cover' }} />
              <button onClick={() => { setImageFile(null); setImagePreview(null); }} style={{
                position: 'absolute', top: 6, right: 6,
                background: 'rgba(0,0,0,0.6)', border: 'none', color: '#fff',
                borderRadius: '50%', width: 24, height: 24, cursor: 'pointer',
                display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 14,
              }}>x</button>
            </div>
          )}
        </div>
      </div>

      {expanded && (
        <>
          <div style={{ borderTop: '1px solid var(--border)', marginTop: 12, paddingTop: 12, display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
            <button onClick={() => fileRef.current?.click()} style={iconActionBtn}>
              <i className="ti ti-photo" style={{ fontSize: 20 }} aria-hidden="true" />
              <span style={{ fontSize: 13 }}>Photo</span>
            </button>
            <input ref={fileRef} type="file" accept=".jpg,.jpeg,.png,.gif" style={{ display: 'none' }} onChange={handleImage} />

            <select value={privacy} onChange={(e) => setPrivacy(e.target.value)} style={selectStyle}>
              <option value="public">🌍 Public</option>
              <option value="followers">👥 Followers</option>
              <option value="selected_followers">🔒 Selected</option>
            </select>

            {error && <span style={{ fontSize: 12, color: 'var(--danger)' }}>{error}</span>}

            <button onClick={submit} disabled={!body.trim() || loading} style={{
              marginLeft: 'auto',
              background: 'var(--accent)', color: '#0d0d0d',
              border: 'none', borderRadius: 8, padding: '8px 20px',
              fontSize: 14, fontWeight: 500, cursor: 'pointer', fontFamily: 'var(--font-body)',
              opacity: (!body.trim() || loading) ? 0.5 : 1,
            }}>
              {loading ? 'Posting...' : 'Post'}
            </button>
          </div>

          {privacy === 'selected_followers' && followers.length > 0 && (
            <div style={{ marginTop: 12, padding: '12px', background: 'var(--bg-elevated)', borderRadius: 8 }}>
              <p style={{ fontSize: 12, color: 'var(--text-muted)', marginBottom: 8 }}>Choose who can see this post:</p>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                {followers.map((f) => (
                  <label key={f.id} style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, cursor: 'pointer', color: 'var(--text-secondary)' }}>
                    <input
                      type="checkbox"
                      checked={viewerIDs.includes(f.id)}
                      onChange={(e) => setViewerIDs((prev) => e.target.checked ? [...prev, f.id] : prev.filter((id) => id !== f.id))}
                      style={{ accentColor: 'var(--accent)' }}
                    />
                    {f.first_name} {f.last_name}
                  </label>
                ))}
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}

/* ─── Post Card ─── */
function PostCard({ post, currentUser, onDeleted }) {
  const [showComments, setShowComments] = useState(false);
  const [comments, setComments] = useState([]);
  const [commentsLoaded, setCommentsLoaded] = useState(false);
  const [commentBody, setCommentBody] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const isOwner = currentUser?.id === post.author_id;

  const toggleComments = async () => {
    setShowComments((s) => !s);
    if (!commentsLoaded) {
      try {
        const data = await getComments(post.id);
        setComments(data.comments ?? []);
        setCommentsLoaded(true);
      } catch {}
    }
  };

  const submitComment = async (e) => {
    e.preventDefault();
    if (!commentBody.trim() || submitting) return;
    setSubmitting(true);
    try {
      const data = await createComment(post.id, { body: commentBody.trim() });
      setComments((prev) => [...prev, data.comment]);
      setCommentBody('');
    } catch {}
    setSubmitting(false);
  };

  const handleDelete = async () => {
    if (!confirm('Delete this post?')) return;
    setDeleting(true);
    try {
      await deletePost(post.id);
      onDeleted(post.id);
    } catch {
      setDeleting(false);
    }
  };

  const author = post.author;
  const avatarSrc = author?.avatar_path ? `/api/proxy${author.avatar_path}` : null;
  const initials = `${author?.first_name?.[0] ?? ''}${author?.last_name?.[0] ?? ''}`.toUpperCase();
  const timeAgo = formatTimeAgo(post.created_at);

  return (
    <div style={{
      background: 'var(--bg-surface)',
      border: '1px solid var(--border)',
      borderRadius: 12,
      marginBottom: 16,
      overflow: 'hidden',
    }}>
      <div style={{ display: 'flex', alignItems: 'center', padding: '12px 16px', gap: 12 }}>
        <Link href={`/profile/${post.author_id}`}>
          <AvatarCircle src={avatarSrc} initials={initials} size={40} />
        </Link>
        <div style={{ flex: 1 }}>
          <Link href={`/profile/${post.author_id}`} style={{ fontSize: 14, fontWeight: 600, color: 'var(--text-primary)', textDecoration: 'none' }}>
            {author?.first_name} {author?.last_name}
          </Link>
          {author?.nickname && (
            <span style={{ fontSize: 12, color: 'var(--text-muted)', marginLeft: 6 }}>@{author.nickname}</span>
          )}
          <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>{timeAgo} · {privacyIcon(post.privacy)}</div>
        </div>
        {isOwner && (
          <button onClick={handleDelete} disabled={deleting} style={{ background: 'none', border: 'none', color: 'var(--text-muted)', cursor: 'pointer', fontSize: 20, padding: 4 }} aria-label="Delete post">
            <i className="ti ti-dots" aria-hidden="true" />
          </button>
        )}
      </div>

      {post.body && (
        <div style={{ padding: '0 16px 12px', fontSize: 15, color: 'var(--text-primary)', lineHeight: 1.6, whiteSpace: 'pre-wrap' }}>
          {post.body}
        </div>
      )}

      {post.image_path && (
        <img src={`/api/proxy${post.image_path}`} alt="" style={{ width: '100%', maxHeight: 500, objectFit: 'cover', display: 'block' }} />
      )}

      <div style={{ padding: '8px 16px', display: 'flex', gap: 20, borderTop: '1px solid var(--border)' }}>
        <button onClick={toggleComments} style={actionBtn}>
          <i className="ti ti-message-circle" style={{ fontSize: 20 }} aria-hidden="true" />
          <span style={{ fontSize: 13, color: 'var(--text-muted)' }}>Comment</span>
        </button>
      </div>

      {showComments && (
        <div style={{ borderTop: '1px solid var(--border)', padding: '12px 16px' }}>
          {!commentsLoaded ? (
            <div style={{ fontSize: 13, color: 'var(--text-muted)' }}>Loading...</div>
          ) : comments.length === 0 ? (
            <div style={{ fontSize: 13, color: 'var(--text-muted)', marginBottom: 12 }}>No comments yet.</div>
          ) : (
            comments.map((c) => <CommentItem key={c.id} comment={c} />)
          )}

          <form onSubmit={submitComment} style={{ display: 'flex', gap: 8, marginTop: 8 }}>
            <input
              value={commentBody}
              onChange={(e) => setCommentBody(e.target.value)}
              placeholder="Add a comment..."
              style={{
                flex: 1, background: 'var(--bg-elevated)', border: '1px solid var(--border)',
                borderRadius: 20, padding: '8px 14px', color: 'var(--text-primary)',
                fontSize: 13, outline: 'none', fontFamily: 'var(--font-body)',
              }}
            />
            <button type="submit" disabled={!commentBody.trim() || submitting} style={{
              background: 'none', border: 'none', color: 'var(--accent)',
              fontSize: 13, fontWeight: 600, cursor: 'pointer', fontFamily: 'var(--font-body)',
              opacity: (!commentBody.trim() || submitting) ? 0.5 : 1,
            }}>Post</button>
          </form>
        </div>
      )}
    </div>
  );
}

function CommentItem({ comment }) {
  const author = comment.author;
  const src = author?.avatar_path ? `/api/proxy${author.avatar_path}` : null;
  const initials = `${author?.first_name?.[0] ?? ''}${author?.last_name?.[0] ?? ''}`.toUpperCase();
  return (
    <div style={{ display: 'flex', gap: 10, marginBottom: 10 }}>
      <AvatarCircle src={src} initials={initials} size={30} />
      <div style={{ background: 'var(--bg-elevated)', borderRadius: 12, padding: '6px 12px', flex: 1 }}>
        <span style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-primary)' }}>
          {author?.first_name} {author?.last_name}
        </span>
        <span style={{ fontSize: 13, color: 'var(--text-secondary)', marginLeft: 8 }}>{comment.body}</span>
      </div>
    </div>
  );
}

function AvatarCircle({ src, initials, size = 36 }) {
  return (
    <div style={{
      width: size, height: size, borderRadius: '50%',
      background: 'var(--bg-elevated)', border: '1px solid var(--border)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      fontSize: size * 0.33, color: 'var(--accent)', fontWeight: 500,
      flexShrink: 0, overflow: 'hidden', cursor: 'pointer',
    }}>
      {src ? <img src={src} alt="" style={{ width: '100%', height: '100%', objectFit: 'cover' }} /> : initials}
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

function formatTimeAgo(dateStr) {
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h`;
  const days = Math.floor(hrs / 24);
  if (days < 7) return `${days}d`;
  return new Date(dateStr).toLocaleDateString();
}

function privacyIcon(privacy) {
  if (privacy === 'public') return '🌍';
  if (privacy === 'followers') return '👥';
  return '🔒';
}

const iconActionBtn = {
  display: 'flex', alignItems: 'center', gap: 6,
  background: 'none', border: 'none', color: 'var(--text-secondary)',
  cursor: 'pointer', fontFamily: 'var(--font-body)', padding: '4px 8px', borderRadius: 8,
};

const actionBtn = {
  display: 'flex', alignItems: 'center', gap: 6,
  background: 'none', border: 'none', color: 'var(--text-secondary)',
  cursor: 'pointer', fontFamily: 'var(--font-body)', padding: '4px 8px', borderRadius: 8,
};

const selectStyle = {
  background: 'var(--bg-elevated)', border: '1px solid var(--border)',
  borderRadius: 8, padding: '6px 10px', color: 'var(--text-secondary)',
  fontSize: 13, fontFamily: 'var(--font-body)', outline: 'none', cursor: 'pointer',
};

const loadMoreBtn = {
  width: '100%', padding: '12px', marginTop: 8,
  background: 'var(--bg-surface)', border: '1px solid var(--border)',
  borderRadius: 10, color: 'var(--text-secondary)', fontSize: 14,
  cursor: 'pointer', fontFamily: 'var(--font-body)',
};
