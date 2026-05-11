'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { useState, useEffect, useRef } from 'react';
import { useAuth } from '../context/AuthContext';
import UserSearchBox from './UserSearchBox';

export default function AppShell({ children }) {
  const { user, logout } = useAuth();
  const router = useRouter();
  const pathname = usePathname();
  const [searchOpen, setSearchOpen] = useState(false);
  const [notifOpen, setNotifOpen] = useState(false);
  const [collapsed, setCollapsed] = useState(false);

  const handleLogout = async () => {
    await logout();
    router.push('/login');
  };

  const isActive = (href) => pathname === href || pathname.startsWith(href + '/');

  const sidebarWidth = collapsed ? 72 : 244;

  return (
    <div style={{ display: 'flex', minHeight: '100vh', background: 'var(--bg)' }}>
      {/* Left Sidebar */}
      <nav style={{
        width: sidebarWidth,
        minHeight: '100vh',
        borderRight: '1px solid var(--border)',
        display: 'flex',
        flexDirection: 'column',
        padding: '12px 0',
        position: 'fixed',
        top: 0,
        left: 0,
        background: 'var(--bg)',
        zIndex: 100,
        transition: 'width 0.2s ease',
        overflow: 'hidden',
      }}>
        {/* Logo */}
        <div style={{ padding: collapsed ? '20px 0 20px 0' : '20px 16px 20px 16px', marginBottom: 8 }}>
          {collapsed ? (
            <div style={{ display: 'flex', justifyContent: 'center' }}>
              <span style={{ fontFamily: 'var(--font-display)', fontSize: 22, color: 'var(--accent)' }}>S</span>
            </div>
          ) : (
            <span style={{ fontFamily: 'var(--font-display)', fontSize: 22, color: 'var(--accent)', letterSpacing: '-0.5px', whiteSpace: 'nowrap' }}>
              Socialite
            </span>
          )}
        </div>

        {/* Nav items */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 4, padding: '0 8px' }}>
          <NavItem href="/feed" icon="ti-home" label="Home" active={isActive('/feed')} collapsed={collapsed} />

          {/* Search — toggles inline panel */}
          <button
            onClick={() => { setSearchOpen(o => !o); setNotifOpen(false); }}
            style={navBtnStyle(searchOpen, collapsed)}
          >
            <i className="ti ti-search" style={{ fontSize: 24, flexShrink: 0 }} aria-hidden="true" />
            {!collapsed && <span style={{ fontSize: 15, fontWeight: searchOpen ? 600 : 400 }}>Search</span>}
          </button>

          <NavItem href="/groups" icon="ti-layout-grid" label="Groups" active={isActive('/groups')} collapsed={collapsed} />
          <NavItem href="/chat" icon="ti-message-circle" label="Messages" active={isActive('/chat')} collapsed={collapsed} />

          {/* Notifications — toggles inline panel */}
          <button
            onClick={() => { setNotifOpen(o => !o); setSearchOpen(false); }}
            style={navBtnStyle(notifOpen, collapsed)}
          >
            <NotifBell collapsed={collapsed} active={notifOpen} />
          </button>

          {/* Create post */}
          <NavItem href="/create" icon="ti-square-plus" label="Create" active={isActive('/create')} collapsed={collapsed} />
          <NavItem href="/profile" icon="ti-user-circle" label="Profile" active={isActive('/profile')} collapsed={collapsed} />
        </div>

        {/* Bottom */}
        <div style={{ padding: '0 8px', display: 'flex', flexDirection: 'column', gap: 4 }}>
          <button
            onClick={() => setCollapsed(c => !c)}
            style={navBtnStyle(false, collapsed)}
            aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          >
            <i className={`ti ${collapsed ? 'ti-layout-sidebar-right' : 'ti-layout-sidebar-left'}`} style={{ fontSize: 24, flexShrink: 0 }} aria-hidden="true" />
            {!collapsed && <span style={{ fontSize: 15 }}>Collapse</span>}
          </button>

          <button onClick={handleLogout} style={navBtnStyle(false, collapsed)}>
            <i className="ti ti-logout" style={{ fontSize: 24, flexShrink: 0 }} aria-hidden="true" />
            {!collapsed && <span style={{ fontSize: 15 }}>Log out</span>}
          </button>
        </div>
      </nav>

      {/* Search panel overlay */}
      {searchOpen && (
        <SearchPanel onClose={() => setSearchOpen(false)} sidebarWidth={sidebarWidth} />
      )}

      {/* Notifications panel overlay */}
      {notifOpen && (
        <NotificationsPanel onClose={() => setNotifOpen(false)} sidebarWidth={sidebarWidth} />
      )}

      {/* Main content */}
      <main style={{
        marginLeft: sidebarWidth,
        flex: 1,
        minHeight: '100vh',
        transition: 'margin-left 0.2s ease',
      }}>
        {children}
      </main>
    </div>
  );
}

function NavItem({ href, icon, label, active, collapsed }) {
  return (
    <Link href={href} style={{
      display: 'flex',
      alignItems: 'center',
      gap: 16,
      padding: collapsed ? '12px 0' : '12px 16px',
      borderRadius: 12,
      color: active ? 'var(--text-primary)' : 'var(--text-secondary)',
      textDecoration: 'none',
      transition: 'background var(--transition)',
      background: active ? 'var(--bg-elevated)' : 'transparent',
      justifyContent: collapsed ? 'center' : 'flex-start',
    }}
      onMouseEnter={e => { if (!active) e.currentTarget.style.background = 'var(--bg-surface)'; }}
      onMouseLeave={e => { if (!active) e.currentTarget.style.background = 'transparent'; }}
    >
      <i className={`ti ${icon}`} style={{ fontSize: 24, flexShrink: 0 }} aria-hidden="true" />
      {!collapsed && <span style={{ fontSize: 15, fontWeight: active ? 600 : 400, whiteSpace: 'nowrap' }}>{label}</span>}
    </Link>
  );
}

function navBtnStyle(active, collapsed) {
  return {
    display: 'flex',
    alignItems: 'center',
    gap: 16,
    padding: collapsed ? '12px 0' : '12px 16px',
    borderRadius: 12,
    color: active ? 'var(--text-primary)' : 'var(--text-secondary)',
    background: active ? 'var(--bg-elevated)' : 'transparent',
    border: 'none',
    cursor: 'pointer',
    width: '100%',
    fontFamily: 'var(--font-body)',
    justifyContent: collapsed ? 'center' : 'flex-start',
    transition: 'background var(--transition)',
  };
}

function NotifBell({ collapsed, active }) {
  const [count, setCount] = useState(0);

  useEffect(() => {
    fetch('/api/proxy/notifications', { credentials: 'include' })
      .then(r => r.json())
      .then(d => setCount(d.unread_count ?? 0))
      .catch(() => {});
  }, []);

  return (
    <>
      <div style={{ position: 'relative', flexShrink: 0 }}>
        <i className="ti ti-bell" style={{ fontSize: 24 }} aria-hidden="true" />
        {count > 0 && (
          <span style={{
            position: 'absolute', top: -4, right: -4,
            background: '#e53935', color: '#fff',
            borderRadius: '50%', fontSize: 10, fontWeight: 700,
            width: 16, height: 16, display: 'flex', alignItems: 'center', justifyContent: 'center',
            lineHeight: 1,
          }}>{count > 9 ? '9+' : count}</span>
        )}
      </div>
      {!collapsed && <span style={{ fontSize: 15, fontWeight: active ? 600 : 400 }}>Notifications</span>}
    </>
  );
}

function SearchPanel({ onClose, sidebarWidth }) {
  return (
    <>
      <div onClick={onClose} style={{ position: 'fixed', inset: 0, zIndex: 98 }} />
      <div style={{
        position: 'fixed',
        left: sidebarWidth,
        top: 0,
        width: 380,
        height: '100vh',
        background: 'var(--bg-surface)',
        borderRight: '1px solid var(--border)',
        zIndex: 99,
        padding: 24,
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
      }}>
        <h2 style={{ fontFamily: 'var(--font-display)', fontSize: 22, color: 'var(--text-primary)', margin: 0 }}>Search</h2>
        <UserSearchBox autoFocus onSelect={onClose} />
      </div>
    </>
  );
}

function NotificationsPanel({ onClose, sidebarWidth }) {
  const [notifs, setNotifs] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch('/api/proxy/notifications', { credentials: 'include' })
      .then(r => r.json())
      .then(d => { setNotifs(d.notifications ?? []); setLoading(false); })
      .catch(() => setLoading(false));
  }, []);

  const markAllRead = async () => {
    await fetch('/api/proxy/notifications/read-all', { method: 'POST', credentials: 'include' });
    setNotifs(prev => prev.map(n => ({ ...n, read: true })));
  };

  return (
    <>
      <div onClick={onClose} style={{ position: 'fixed', inset: 0, zIndex: 98 }} />
      <div style={{
        position: 'fixed',
        left: sidebarWidth,
        top: 0,
        width: 380,
        height: '100vh',
        background: 'var(--bg-surface)',
        borderRight: '1px solid var(--border)',
        zIndex: 99,
        display: 'flex',
        flexDirection: 'column',
      }}>
        <div style={{ padding: '24px 24px 12px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid var(--border)' }}>
          <h2 style={{ fontFamily: 'var(--font-display)', fontSize: 22, color: 'var(--text-primary)', margin: 0 }}>Notifications</h2>
          <button onClick={markAllRead} style={{ background: 'none', border: 'none', color: 'var(--accent)', fontSize: 12, cursor: 'pointer', fontFamily: 'var(--font-body)' }}>
            Mark all read
          </button>
        </div>

        <div style={{ flex: 1, overflowY: 'auto' }}>
          {loading ? (
            <div style={{ padding: 24, color: 'var(--text-muted)', fontSize: 13 }}>Loading…</div>
          ) : notifs.length === 0 ? (
            <div style={{ padding: 24, color: 'var(--text-muted)', fontSize: 13 }}>No notifications yet.</div>
          ) : (
            notifs.map(n => <NotifItem key={n.id} notif={n} onClose={onClose} />)
          )}
        </div>
      </div>
    </>
  );
}

function NotifItem({ notif, onClose }) {
  const router = useRouter();
  const label = {
    follow_request: 'sent you a follow request',
    group_invitation: `invited you to join a group`,
    group_join_request: 'requested to join your group',
    group_event: 'created a new event in',
  }[notif.type] ?? '';

  const handleClick = () => {
    if (notif.type === 'follow_request') router.push('/profile');
    else if (notif.group) router.push(`/groups/${notif.group.id}`);
    onClose();
  };

  return (
    <div onClick={handleClick} style={{
      display: 'flex',
      gap: 12,
      padding: '12px 20px',
      cursor: 'pointer',
      background: notif.read ? 'transparent' : 'rgba(201,185,154,0.06)',
      borderBottom: '1px solid var(--border)',
      transition: 'background var(--transition)',
    }}
      onMouseEnter={e => e.currentTarget.style.background = 'var(--bg-elevated)'}
      onMouseLeave={e => e.currentTarget.style.background = notif.read ? 'transparent' : 'rgba(201,185,154,0.06)'}
    >
      <Avatar user={notif.actor} size={40} />
      <div style={{ flex: 1, fontSize: 13, color: 'var(--text-secondary)', lineHeight: 1.5 }}>
        <span style={{ color: 'var(--text-primary)', fontWeight: 500 }}>
          {notif.actor ? `${notif.actor.first_name} ${notif.actor.last_name}` : 'Someone'}
        </span>{' '}
        {label}
        {notif.group && <span style={{ color: 'var(--accent)' }}> {notif.group.title}</span>}
        <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 2 }}>
          {new Date(notif.created_at).toLocaleDateString()}
        </div>
      </div>
      {!notif.read && (
        <div style={{ width: 8, height: 8, borderRadius: '50%', background: 'var(--accent)', flexShrink: 0, marginTop: 4 }} />
      )}
    </div>
  );
}

function Avatar({ user, size = 36 }) {
  if (!user) return (
    <div style={{
      width: size, height: size, borderRadius: '50%',
      background: 'var(--bg-elevated)', border: '1px solid var(--border)',
      flexShrink: 0,
    }} />
  );
  const initials = `${user.first_name?.[0] ?? ''}${user.last_name?.[0] ?? ''}`.toUpperCase();
  const src = user.avatar_path ? `/api/proxy${user.avatar_path}` : null;
  return (
    <div style={{
      width: size, height: size, borderRadius: '50%',
      background: 'var(--bg-elevated)', border: '1px solid var(--border)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      fontSize: size * 0.3, color: 'var(--accent)', fontWeight: 500,
      flexShrink: 0, overflow: 'hidden',
    }}>
      {src ? <img src={src} alt="" style={{ width: '100%', height: '100%', objectFit: 'cover' }} /> : initials}
    </div>
  );
}
