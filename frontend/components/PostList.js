'use client';

import PostCard from './PostCard';

export default function PostList({ posts }) {
  if (!posts?.length) {
    return (
      <div style={{
        background: 'var(--bg-surface)',
        border: '1px solid var(--border)',
        borderRadius: 'var(--radius-md)',
        padding: '24px',
        color: 'var(--text-secondary)',
        fontSize: 14,
      }}>
        No posts yet.
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
      {posts.map((post) => (
        <PostCard key={post.id} post={post} />
      ))}
    </div>
  );
}

