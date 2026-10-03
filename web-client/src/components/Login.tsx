import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { Layers } from 'lucide-react';

export function Login() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');
    try {
      const res = await fetch('/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password })
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        let errorMsg = data.error?.message || data.message || 'Login failed';
        if (errorMsg.includes('internal server error') || errorMsg.includes('database')) {
          errorMsg = 'Our system is experiencing a temporary issue. Please try again later.';
        } else if (errorMsg.includes('credentials') || errorMsg.includes('UNAUTHENTICATED')) {
          errorMsg = 'Incorrect email or password.';
        } else {
          errorMsg = 'Unable to sign in. Please check your details and try again.';
        }
        throw new Error(errorMsg);
      }
      localStorage.setItem('token', data.access_token);
      navigate('/dashboard');
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="auth-layout">
      <main className="auth-card">
        
        <div className="auth-logo">
          <Layers size={24} />
          Ledger
        </div>

        <div className="auth-header">
          <h1>Sign in</h1>
          <p>Enter your details to access your dashboard.</p>
        </div>
        
        {error && (
          <div className="error-banner">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Email</label>
            <input 
              type="email" 
              placeholder="you@example.com" 
              required 
              value={email} 
              onChange={e => setEmail(e.target.value)} 
              disabled={loading}
              autoComplete="email"
            />
          </div>

          <div className="form-group">
            <label>Password</label>
            <input 
              type="password" 
              placeholder="••••••••" 
              required 
              value={password} 
              onChange={e => setPassword(e.target.value)} 
              disabled={loading}
              autoComplete="current-password"
            />
          </div>

          <button type="submit" disabled={loading}>
            {loading ? 'Signing in...' : 'Sign in'}
          </button>
        </form>
        
        <div className="auth-footer">
          Don't have an account? <Link to="/register">Sign up</Link>
        </div>
      </main>
    </div>
  );
}
