import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';

export function Login() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const res = await fetch('/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password })
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error?.message || 'Login failed');
      
      localStorage.setItem('token', data.access_token);
      navigate('/dashboard');
    } catch (err: any) {
      setError(err.message);
    }
  };

  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: '100vh', background: 'var(--bg-canvas)' }}>
      <main className="glass-card" style={{ width: '100%', maxWidth: '380px', padding: '32px' }}>
        <div style={{ textAlign: 'center', marginBottom: '24px' }}>
          <h1 style={{ fontSize: '1.5rem', marginBottom: '8px', color: 'var(--text-primary)' }}>Welcome back</h1>
          <p style={{ color: 'var(--text-muted)', fontSize: '0.875rem', margin: 0 }}>Enter your details to access your account.</p>
        </div>
        
        {error && (
          <div style={{ background: 'var(--status-danger-bg)', border: '1px solid var(--status-danger)', color: 'var(--status-danger)', padding: '12px', borderRadius: '8px', margin: '0 0 20px 0', fontSize: '0.875rem' }}>
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} style={{ display: 'grid', gap: '16px' }}>
          <div>
            <label style={{ display: 'block', fontSize: '0.875rem', fontWeight: 500, marginBottom: '6px' }}>Email</label>
            <input type="email" placeholder="you@example.com" required value={email} onChange={e => setEmail(e.target.value)} />
          </div>
          <div>
            <label style={{ display: 'block', fontSize: '0.875rem', fontWeight: 500, marginBottom: '6px' }}>Password</label>
            <input type="password" placeholder="••••••••" required value={password} onChange={e => setPassword(e.target.value)} />
          </div>
          <button type="submit" style={{ marginTop: '8px', width: '100%' }}>Sign In</button>
        </form>
        
        <p style={{ marginTop: '24px', textAlign: 'center', color: 'var(--text-muted)', fontSize: '0.875rem' }}>
          Don't have an account? <Link to="/register" style={{ fontWeight: 500, color: 'var(--action-primary)' }}>Sign up</Link>
        </p>
      </main>
    </div>
  );
}
