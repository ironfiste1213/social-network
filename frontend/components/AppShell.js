'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useAuth } from '../context/AuthContext';
import UserSearchBox from './UserSearchBox';

export default function AppShell({ children }) {
  const { user, logout } = useAuth();
  const router = useRouter();

  const handleLogout = async () => {
    await logout();
    router.push('/login');
  };

  return (
    <div style={{ minHeight: '100vh', background: 'var(--bg)', display: 'flex', flexDirection: 'column' }}>
      {/* Nav */}
      <nav style={{
        height: 56,
        borderBottom: '1px solid var(--border)',
        display: 'flex',
        alignItems: 'center',
        padding: '0 32px',
        gap: 32,
        position: 'sticky',
        top: 0,
        background: 'var(--bg)',
        zIndex: 100,
      }}>
        <Link href="/feed" style={{
          fontFamily: 'var(--font-display)',
          fontSize: 18,
          color: 'var(--accent)',
          letterSpacing: '-0.5px',
          flexShrink: 0,
        }}>
          Socialite
        </Link>

        <div style={{ flex: 1, display: 'flex', justifyContent: 'center', padding: '0 20px' }}>
          {user && <UserSearchBox />}
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: 24 }}>
          <NavLink href="/feed">Feed</NavLink>
          <NavLink href="/profile">Profile</NavLink>

          <button
            onClick={handleLogout}
            style={{
              background: 'none',
              border: '1px solid var(--border)',
              borderRadius: 'var(--radius-sm)',
              padding: '6px 14px',
              color: 'var(--text-secondary)',
              fontSize: 13,
              cursor: 'pointer',
              transition: 'color var(--transition), border-color var(--transition)',
            }}
            onMouseEnter={(e) => { e.currentTarget.style.color = 'var(--text-primary)'; e.currentTarget.style.borderColor = 'var(--border-focus)'; }}
            onMouseLeave={(e) => { e.currentTarget.style.color = 'var(--text-secondary)'; e.currentTarget.style.borderColor = 'var(--border)'; }}
          >
            Sign out
          </button>

          {user && (
            <Link href="/profile" style={{
              width: 32,
              height: 32,
              borderRadius: '50%',
              background: 'var(--bg-elevated)',
              border: '1px solid var(--border)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: 13,
              color: 'var(--accent)',
              fontWeight: 500,
            }}>
              {user.first_name?.[0]?.toUpperCase() ?? '?'}
            </Link>
          )}
        </div>
      </nav>

      {/* Page content */}
      <main style={{ flex: 1 }}>
        {children}
      </main>
    </div>
  );
}

function NavLink({ href, children }) {
  return (
    <Link
      href={href}
      style={{ fontSize: 14, color: 'var(--text-secondary)', transition: 'color var(--transition)' }}
      onMouseEnter={(e) => e.currentTarget.style.color = 'var(--text-primary)'}
      onMouseLeave={(e) => e.currentTarget.style.color = 'var(--text-secondary)'}
    >
      {children}
    </Link>
  );
}
