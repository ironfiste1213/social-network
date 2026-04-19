'use client';

import AuthGuard from '../../components/AuthGuard';
import AppShell from '../../components/AppShell';
import { useAuth } from '../../context/AuthContext';

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
      <p style={{ fontSize: 14, color: 'var(--text-secondary)', marginBottom: 40 }}>
        Your feed is empty for now — posts are coming on Day 4.
      </p>

      {/* Placeholder cards */}
      {[...Array(3)].map((_, i) => (
        <div key={i} style={{
          background: 'var(--bg-surface)',
          border: '1px solid var(--border)',
          borderRadius: 'var(--radius-md)',
          padding: '20px 24px',
          marginBottom: 16,
          opacity: 1 - i * 0.2,
        }}>
          <div style={{ display: 'flex', gap: 12, alignItems: 'center', marginBottom: 16 }}>
            <div style={{
              width: 36, height: 36, borderRadius: '50%',
              background: 'var(--bg-elevated)', border: '1px solid var(--border)',
            }} />
            <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
              <div style={{ width: 100, height: 10, background: 'var(--bg-elevated)', borderRadius: 4 }} />
              <div style={{ width: 60, height: 8, background: 'var(--bg-elevated)', borderRadius: 4 }} />
            </div>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            <div style={{ width: '100%', height: 10, background: 'var(--bg-elevated)', borderRadius: 4 }} />
            <div style={{ width: '80%', height: 10, background: 'var(--bg-elevated)', borderRadius: 4 }} />
            <div style={{ width: '60%', height: 10, background: 'var(--bg-elevated)', borderRadius: 4 }} />
          </div>
        </div>
      ))}
    </div>
  );
}