'use client';

import { useState, useEffect, useRef } from 'react';
import AuthGuard from '../../components/AuthGuard';
import AppShell from '../../components/AppShell';
import { useAuth } from '../../context/AuthContext';
import { getConversations, getPrivateHistory, getGroupHistory, createChatSocket } from '../../services/chat';

export default function ChatPage() {
  return (
    <AuthGuard>
      <AppShell>
        <ChatContent />
      </AppShell>
    </AuthGuard>
  );
}

function ChatContent() {
  const { user } = useAuth();
  const [conversations, setConversations] = useState([]);
  const [selected, setSelected] = useState(null); // { type, id, title, participant }
  const [messages, setMessages] = useState([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(true);
  const [msgLoading, setMsgLoading] = useState(false);
  const wsRef = useRef(null);
  const bottomRef = useRef(null);
  const inputRef = useRef(null);

  useEffect(() => {
    getConversations()
      .then((d) => setConversations(d.conversations ?? []))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (!user) return;
    const ws = createChatSocket();
    wsRef.current = ws;

    ws.onmessage = (e) => {
      try {
        const event = JSON.parse(e.data);
        if (event.type === 'message' && event.payload) {
          const msg = event.payload;
          setMessages((prev) => {
            if (!selected) return prev;
            const isPrivate = msg.chat_type === 'private';
            if (isPrivate && selected.type === 'private') return [...prev, msg];
            if (!isPrivate && selected.type === 'group' && msg.chat_id === selected.id) return [...prev, msg];
            return prev;
          });

          setConversations((prev) => prev.map((c) => {
            const match = c.chat_id === msg.chat_id;
            return match ? { ...c, last_message: msg.body, last_at: msg.created_at } : c;
          }));
        }
      } catch {}
    };

    ws.onerror = () => {};
    ws.onclose = () => {};

    return () => ws.close();
  }, [user, selected]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const selectConversation = async (conv) => {
    const isGroup = conv.chat_type === 'group';
    setSelected({
      type: isGroup ? 'group' : 'private',
      id: isGroup ? conv.chat_id : conv.participant?.id,
      title: isGroup ? conv.group_title : `${conv.participant?.first_name} ${conv.participant?.last_name}`,
      participant: conv.participant,
    });
    setMessages([]);
    setMsgLoading(true);
    try {
      const data = isGroup
        ? await getGroupHistory(conv.chat_id)
        : await getPrivateHistory(conv.participant?.id);
      setMessages(data.messages ?? []);
    } catch {}
    setMsgLoading(false);
    inputRef.current?.focus();
  };

  const sendMessage = (e) => {
    e.preventDefault();
    if (!input.trim() || !selected || !wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;
    const type = selected.type === 'group' ? 'send_group' : 'send_private';
    wsRef.current.send(JSON.stringify({ type, to: selected.id, body: input.trim() }));
    setInput('');
  };

  const initials = (u) => `${u?.first_name?.[0] ?? ''}${u?.last_name?.[0] ?? ''}`.toUpperCase();

  return (
    <div style={{ display: 'flex', height: '100vh', overflow: 'hidden' }}>
      <div style={{
        width: 360,
        borderRight: '1px solid var(--border)',
        display: 'flex',
        flexDirection: 'column',
        background: 'var(--bg)',
        flexShrink: 0,
      }}>
        <div style={{ padding: '24px 20px 16px', borderBottom: '1px solid var(--border)' }}>
          <h1 style={{ fontFamily: 'var(--font-display)', fontSize: 22, color: 'var(--text-primary)', margin: 0 }}>
            Messages
          </h1>
        </div>

        <div style={{ flex: 1, overflowY: 'auto' }}>
          {loading ? (
            <div style={{ padding: 24 }}><Spinner /></div>
          ) : conversations.length === 0 ? (
            <div style={{ padding: '24px 20px', fontSize: 14, color: 'var(--text-muted)' }}>
              No conversations yet. Follow someone and start chatting!
            </div>
          ) : (
            conversations.map((conv, i) => {
              const isGroup = conv.chat_type === 'group';
              const title = isGroup ? conv.group_title : `${conv.participant?.first_name} ${conv.participant?.last_name}`;
              const isSelected = selected && (
                (isGroup && selected.id === conv.chat_id) ||
                (!isGroup && selected.id === conv.participant?.id)
              );

              return (
                <div
                  key={conv.chat_id || i}
                  onClick={() => selectConversation(conv)}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 12,
                    padding: '12px 20px',
                    cursor: 'pointer',
                    background: isSelected ? 'var(--bg-elevated)' : 'transparent',
                    borderBottom: '1px solid var(--border)',
                    transition: 'background var(--transition)',
                  }}
                  onMouseEnter={(e) => { if (!isSelected) e.currentTarget.style.background = 'var(--bg-surface)'; }}
                  onMouseLeave={(e) => { if (!isSelected) e.currentTarget.style.background = 'transparent'; }}
                >
                  <ConvAvatar conv={conv} isGroup={isGroup} initials={initials(conv.participant)} />
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: 14, fontWeight: 500, color: 'var(--text-primary)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                      {title}
                    </div>
                    <div style={{ fontSize: 12, color: 'var(--text-muted)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', marginTop: 2 }}>
                      {conv.last_message || 'Start a conversation'}
                    </div>
                  </div>
                </div>
              );
            })
          )}
        </div>
      </div>

      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        {!selected ? (
          <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', flexDirection: 'column', gap: 12, color: 'var(--text-muted)' }}>
            <i className="ti ti-message-circle" style={{ fontSize: 48, color: 'var(--border)' }} aria-hidden="true" />
            <p style={{ fontSize: 15, margin: 0 }}>Select a conversation</p>
          </div>
        ) : (
          <>
            <div style={{ padding: '16px 24px', borderBottom: '1px solid var(--border)', display: 'flex', alignItems: 'center', gap: 12, background: 'var(--bg)', flexShrink: 0 }}>
              <div style={{ width: 40, height: 40, borderRadius: '50%', background: 'var(--bg-elevated)', border: '1px solid var(--border)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 14, color: 'var(--accent)', fontWeight: 500, overflow: 'hidden' }}>
                {selected.participant?.avatar_path
                  ? <img src={`/api/proxy${selected.participant.avatar_path}`} alt="" style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
                  : selected.type === 'group'
                  ? <i className="ti ti-users" style={{ fontSize: 18 }} aria-hidden="true" />
                  : initials(selected.participant)
                }
              </div>
              <div>
                <div style={{ fontSize: 15, fontWeight: 600, color: 'var(--text-primary)' }}>{selected.title}</div>
                <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>{selected.type === 'group' ? 'Group chat' : 'Direct message'}</div>
              </div>
            </div>

            <div style={{ flex: 1, overflowY: 'auto', padding: '16px 24px', display: 'flex', flexDirection: 'column', gap: 4 }}>
              {msgLoading ? (
                <div style={{ display: 'flex', justifyContent: 'center', paddingTop: 40 }}><Spinner /></div>
              ) : messages.length === 0 ? (
                <div style={{ textAlign: 'center', color: 'var(--text-muted)', fontSize: 14, paddingTop: 40 }}>
                  No messages yet. Say hello!
                </div>
              ) : (
                messages.map((msg, i) => {
                  const isMine = msg.sender_id === user?.id;
                  const showAvatar = !isMine && (i === 0 || messages[i - 1]?.sender_id !== msg.sender_id);
                  const showName = showAvatar;

                  return (
                    <div key={msg.id} style={{ display: 'flex', flexDirection: isMine ? 'row-reverse' : 'row', alignItems: 'flex-end', gap: 8, marginBottom: 2 }}>
                      {!isMine && (
                        <div style={{ width: 28, height: 28, borderRadius: '50%', background: 'var(--bg-elevated)', border: '1px solid var(--border)', flexShrink: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 10, color: 'var(--accent)', overflow: 'hidden', opacity: showAvatar ? 1 : 0 }}>
                          {msg.sender?.avatar_path
                            ? <img src={`/api/proxy${msg.sender.avatar_path}`} alt="" style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
                            : initials(msg.sender)
                          }
                        </div>
                      )}
                      <div style={{ maxWidth: '65%', display: 'flex', flexDirection: 'column', alignItems: isMine ? 'flex-end' : 'flex-start' }}>
                        {showName && (
                          <span style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 2, paddingLeft: 4 }}>
                            {msg.sender?.first_name} {msg.sender?.last_name}
                          </span>
                        )}
                        <div style={{
                          background: isMine ? 'var(--accent)' : 'var(--bg-elevated)',
                          color: isMine ? '#0d0d0d' : 'var(--text-primary)',
                          borderRadius: isMine ? '18px 18px 4px 18px' : '18px 18px 18px 4px',
                          padding: '9px 14px',
                          fontSize: 14,
                          lineHeight: 1.4,
                          wordBreak: 'break-word',
                        }}>
                          {msg.body}
                        </div>
                        <span style={{ fontSize: 10, color: 'var(--text-muted)', marginTop: 2, paddingLeft: 4 }}>
                          {new Date(msg.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                        </span>
                      </div>
                    </div>
                  );
                })
              )}
              <div ref={bottomRef} />
            </div>

            <form onSubmit={sendMessage} style={{
              padding: '12px 24px 20px',
              borderTop: '1px solid var(--border)',
              display: 'flex',
              gap: 10,
              alignItems: 'center',
              background: 'var(--bg)',
              flexShrink: 0,
            }}>
              <input
                ref={inputRef}
                value={input}
                onChange={(e) => setInput(e.target.value)}
                placeholder="Message..."
                style={{
                  flex: 1,
                  background: 'var(--bg-elevated)',
                  border: '1px solid var(--border)',
                  borderRadius: 24,
                  padding: '10px 18px',
                  color: 'var(--text-primary)',
                  fontSize: 14,
                  fontFamily: 'var(--font-body)',
                  outline: 'none',
                }}
              />
              <button
                type="submit"
                disabled={!input.trim()}
                style={{
                  width: 40, height: 40, borderRadius: '50%',
                  background: input.trim() ? 'var(--accent)' : 'var(--bg-elevated)',
                  border: '1px solid var(--border)',
                  color: input.trim() ? '#0d0d0d' : 'var(--text-muted)',
                  cursor: input.trim() ? 'pointer' : 'default',
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  transition: 'background var(--transition)',
                  flexShrink: 0,
                }}
                aria-label="Send"
              >
                <i className="ti ti-send" style={{ fontSize: 18 }} aria-hidden="true" />
              </button>
            </form>
          </>
        )}
      </div>
    </div>
  );
}

function ConvAvatar({ conv, isGroup, initials }) {
  if (isGroup) return (
    <div style={{ width: 44, height: 44, borderRadius: '50%', background: 'var(--bg-elevated)', border: '1px solid var(--border)', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
      <i className="ti ti-users" style={{ fontSize: 20, color: 'var(--accent)' }} aria-hidden="true" />
    </div>
  );
  const src = conv.participant?.avatar_path ? `/api/proxy${conv.participant.avatar_path}` : null;
  return (
    <div style={{ width: 44, height: 44, borderRadius: '50%', background: 'var(--bg-elevated)', border: '1px solid var(--border)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 14, color: 'var(--accent)', fontWeight: 500, flexShrink: 0, overflow: 'hidden' }}>
      {src ? <img src={src} alt="" style={{ width: '100%', height: '100%', objectFit: 'cover' }} /> : initials}
    </div>
  );
}

function Spinner() {
  return (
    <div style={{ display: 'flex', justifyContent: 'center', padding: 20 }}>
      <div style={{ width: 22, height: 22, borderRadius: '50%', border: '2px solid var(--border)', borderTopColor: 'var(--accent)', animation: 'spin 0.8s linear infinite' }} />
      <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
    </div>
  );
}
