'use client';

import { useEffect, useState } from 'react';
import AuthGuard from '../../components/AuthGuard';
import AppShell from '../../components/AppShell';
import PostList from '../../components/PostList';
import { useAuth } from '../../context/AuthContext';
import { getFeedPosts } from '../../services/posts';

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
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState('');
  const [hasMore, setHasMore] = useState(true);

  useEffect(() => {
    let ignore = false;

    async function loadFirstPage() {
      setLoading(true);
      setError('');
      try {
        const data = await getFeedPosts({ limit: 20 });
        if (ignore) return;
        const nextPosts = data?.posts || [];
        setPosts(nextPosts);
        setHasMore(nextPosts.length === 20);
      } catch (err) {
        if (ignore) return;
        setError(err.message || 'Failed to load feed');
      } finally {
        if (!ignore) setLoading(false);
      }
    }

    loadFirstPage();
    return () => {
      ignore = true;
    };
  }, []);

  async function handleLoadMore() {
    if (!posts.length || loadingMore || !hasMore) return;
    const beforeId = posts[posts.length - 1]?.id;
    if (!beforeId) return;

    setLoadingMore(true);
    setError('');
    try {
      const data = await getFeedPosts({ limit: 20, beforeId });
      const incoming = data?.posts || [];
      setPosts((prev) => [...prev, ...incoming]);
      setHasMore(incoming.length === 20);
    } catch (err) {
      setError(err.message || 'Failed to load more posts');
    } finally {
      setLoadingMore(false);
    }
  }

  return (
    <div style={{
      maxWidth: 640,
      margin: '0 auto',
      padding: '48px 24px',
    }}>
      <h2 style={{
        fontFamily: 'var(--font-display)',
        fontSize: 28,
        letterSpacing: '-0.5px',
        marginBottom: 8,
        color: 'var(--text-primary)',
      }}>
        Good to see you, {user?.first_name}.
      </h2>
      <p style={{ fontSize: 14, color: 'var(--text-secondary)', marginBottom: 20 }}>
        Latest posts from your network.
      </p>

      {error ? (
        <div style={{
          marginBottom: 16,
          background: 'rgba(192, 87, 74, 0.1)',
          border: '1px solid rgba(192, 87, 74, 0.3)',
          borderRadius: 'var(--radius-sm)',
          padding: '10px 14px',
          fontSize: 13,
          color: '#e87060',
        }}>
          {error}
        </div>
      ) : null}

      {loading ? (
        <div style={{ color: 'var(--text-secondary)', fontSize: 14 }}>Loading feed...</div>
      ) : (
        <>
          <PostList posts={posts} />
          {hasMore ? (
            <button
              onClick={handleLoadMore}
              disabled={loadingMore}
              style={{
                marginTop: 16,
                width: '100%',
                background: 'var(--bg-surface)',
                border: '1px solid var(--border)',
                borderRadius: 'var(--radius-sm)',
                color: 'var(--text-primary)',
                fontSize: 14,
                padding: '10px 14px',
                opacity: loadingMore ? 0.7 : 1,
              }}
            >
              {loadingMore ? 'Loading...' : 'Load more'}
            </button>
          ) : null}
        </>
      )}
    </div>
  );
}
