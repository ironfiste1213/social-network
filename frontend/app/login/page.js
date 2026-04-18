'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { useAuth } from '../../context/AuthContext';

export default function LoginPage() {
  const { login } = useAuth();
  const router = useRouter();

  const [form, setForm] = useState({ email: '', password: '' });
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const set = (k) => (e) => setForm((f) => ({ ...f, [k]: e.target.value }));

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    const result = await login(form);
    setLoading(false);
    if (result.success) {
      router.push('/feed');
    } else {
      setError(result.error);
    }
  };

  return (
    <div className="auth-page">
      <span className="auth-brand">Socialite</span>

      {/* Form panel */}
      <div className="auth-panel">
        <h1 className="form-heading">
          Welcome<br />back.
        </h1>
        <p className="form-subheading">
          No account yet?{' '}
          <Link href="/register">Create one</Link>
        </p>

        <form className="form-grid" onSubmit={handleSubmit}>
          {error && <div className="form-error">{error}</div>}

          <div className="field">
            <label htmlFor="email">Email</label>
            <input
              id="email"
              type="email"
              autoComplete="email"
              placeholder="you@example.com"
              value={form.email}
              onChange={set('email')}
              required
            />
          </div>

          <div className="field">
            <label htmlFor="password">Password</label>
            <input
              id="password"
              type="password"
              autoComplete="current-password"
              placeholder="••••••••"
              value={form.password}
              onChange={set('password')}
              required
            />
          </div>

          <button className="btn-primary" type="submit" disabled={loading}>
            {loading ? 'Signing in…' : 'Sign in'}
          </button>
        </form>
      </div>

      {/* Decorative panel */}
      <div className="auth-deco">
        <div className="auth-deco-inner" />
        <p className="auth-deco-quote">
          "The people who are crazy enough to think they can change the world are the ones who do."
        </p>
      </div>
    </div>
  );
}