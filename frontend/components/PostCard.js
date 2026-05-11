'use client';

function formatDate(value) {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  return date.toLocaleString();
}

export default function PostCard({ post }) {
  const author = post?.author;
  const fullName = author ? `${author.first_name || ''} ${author.last_name || ''}`.trim() : 'Unknown User';
  const handle = author?.nickname ? `@${author.nickname}` : '';

  return (
    <article style={{
      background: 'var(--bg-surface)',
      border: '1px solid var(--border)',
      borderRadius: 'var(--radius-md)',
      padding: '18px 20px',
      display: 'flex',
      flexDirection: 'column',
      gap: 12,
    }}>
      <header style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <div style={{
            width: 34,
            height: 34,
            borderRadius: '50%',
            border: '1px solid var(--border)',
            background: 'var(--bg-elevated)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: 'var(--accent)',
            fontWeight: 500,
            fontSize: 13,
          }}>
            {(author?.first_name?.[0] || '?').toUpperCase()}
          </div>
          <div style={{ display: 'flex', flexDirection: 'column' }}>
            <strong style={{ fontSize: 14, fontWeight: 500 }}>{fullName}</strong>
            <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>
              {handle} {handle ? '• ' : ''}{formatDate(post?.created_at)}
            </span>
          </div>
        </div>
        <span style={{ fontSize: 11, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.06em' }}>
          {post?.privacy || 'public'}
        </span>
      </header>

      <p style={{ fontSize: 14, lineHeight: 1.6, color: 'var(--text-primary)', whiteSpace: 'pre-wrap' }}>
        {post?.body}
      </p>

      {post?.image_path ? (
        <img
          src={post.image_path}
          alt="Post media"
          style={{ width: '100%', borderRadius: 'var(--radius-sm)', border: '1px solid var(--border)', objectFit: 'cover' }}
        />
      ) : null}
    </article>
  );
}

