'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { useAuth } from '../../context/AuthContext';

const INITIAL = {
  email: '', password: '', confirm: '',
  first_name: '', last_name: '',
  date_of_birth: '',
  nickname: '', about_me: '',
};

export default function RegisterPage() {
  const { register } = useAuth();
  const router = useRouter();

  const [form, setForm]     = useState(INITIAL);
  const [errors, setErrors] = useState({});
  const [apiError, setApiError] = useState('');
  const [loading, setLoading]   = useState(false);

  const set = (k) => (e) => {
    setForm((f) => ({ ...f, [k]: e.target.value }));
    if (errors[k]) setErrors((e) => ({ ...e, [k]: '' }));
  };

  const validate = () => {
    const next = {};
    if (!form.email.includes('@'))      next.email      = 'Enter a valid email.';
    if (form.password.length < 6)       next.password   = 'At least 6 characters.';
    if (form.password !== form.confirm) next.confirm    = 'Passwords do not match.';
    if (!form.first_name.trim())        next.first_name = 'Required.';
    if (!form.last_name.trim())         next.last_name  = 'Required.';
    if (!form.date_of_birth)            next.date_of_birth = 'Required.';
    return next;
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    const errs = validate();
    if (Object.keys(errs).length) { setErrors(errs); return; }

    setApiError('');
    setLoading(true);

    const { confirm, ...payload } = form;          // strip confirm field
    const result = await register(payload);
    setLoading(false);

    if (result.success) {
      router.push('/feed');
    } else {
      setApiError(result.error);
    }
  };

  const field = (name, label, props = {}) => (
    <div className="field">
      <label htmlFor={name}>{label}</label>
      <input
        id={name}
        name={name}
        value={form[name]}
        onChange={set(name)}
        className={errors[name] ? 'error' : ''}
        {...props}
      />
      {errors[name] && <span className="field-error">{errors[name]}</span>}
    </div>
  );

  return (
    <div className="auth-page">
      <span className="auth-brand">Socialite</span>

      {/* Form panel */}
      <div className="auth-panel" style={{ overflowY: 'auto', maxHeight: '100vh' }}>
        <h1 className="form-heading">
          Join the<br />network.
        </h1>
        <p className="form-subheading">
          Already a member?{' '}
          <Link href="/login">Sign in</Link>
        </p>

        <form className="form-grid" onSubmit={handleSubmit} noValidate>
          {apiError && <div className="form-error">{apiError}</div>}

          {/* Name row */}
          <div className="form-row">
            {field('first_name', 'First name', { placeholder: 'Jane', autoComplete: 'given-name' })}
            {field('last_name',  'Last name',  { placeholder: 'Smith', autoComplete: 'family-name' })}
          </div>

          {field('email', 'Email', { type: 'email', placeholder: 'jane@example.com', autoComplete: 'email' })}

          <div className="form-row">
            {field('password', 'Password', { type: 'password', placeholder: '••••••••', autoComplete: 'new-password' })}
            {field('confirm',  'Confirm password', { type: 'password', placeholder: '••••••••', autoComplete: 'new-password' })}
          </div>

          {field('date_of_birth', 'Date of birth', { type: 'date', autoComplete: 'bday' })}

          {/* Optional section */}
          <div className="divider">optional</div>

          {field('nickname', 'Nickname', { placeholder: 'How friends call you' })}

          <div className="field">
            <label htmlFor="about_me">About me</label>
            <textarea
              id="about_me"
              name="about_me"
              value={form.about_me}
              onChange={set('about_me')}
              placeholder="A short bio…"
              rows={3}
              style={{
                background: 'var(--bg-input)',
                border: '1px solid var(--border)',
                borderRadius: 'var(--radius-sm)',
                padding: '11px 14px',
                color: 'var(--text-primary)',
                resize: 'vertical',
                outline: 'none',
                transition: 'border-color var(--transition), box-shadow var(--transition)',
              }}
              onFocus={(e) => {
                e.target.style.borderColor = 'var(--border-focus)';
                e.target.style.boxShadow   = '0 0 0 3px var(--accent-glow)';
              }}
              onBlur={(e) => {
                e.target.style.borderColor = 'var(--border)';
                e.target.style.boxShadow   = 'none';
              }}
            />
          </div>

          <p className="form-optional">Avatar upload available after registration in your profile settings.</p>

          <button className="btn-primary" type="submit" disabled={loading}>
            {loading ? 'Creating account…' : 'Create account'}
          </button>
        </form>
      </div>

      {/* Decorative panel */}
      <div className="auth-deco">
        <div className="auth-deco-inner" />
        <p className="auth-deco-quote">
          "Alone we can do so little;<br />together we can do so much."
        </p>
      </div>
    </div>
  );
}