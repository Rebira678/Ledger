import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { Layers } from 'lucide-react';

export function Register() {
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
      const res = await fetch('/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password })
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        let errorMsg = data.error?.message || data.message || 'Registration failed';
        if (errorMsg.includes('internal server error') || errorMsg.includes('database')) {
          errorMsg = 'Our system is experiencing a temporary issue. Please try again later.';
        } else if (errorMsg.includes('already registered') || errorMsg.includes('duplicate')) {
          errorMsg = 'An account with this email already exists.';
        } else if (errorMsg.includes('password must be at least')) {
          errorMsg = 'Your password must be at least 8 characters long.';
        } else {
          errorMsg = 'Unable to create your account. Please check your details and try again.';
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
          <h1>Create an account</h1>
          <p>Join Ledger to start automating your finances.</p>
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
              minLength={8}
              value={password} 
              onChange={e => setPassword(e.target.value)} 
              disabled={loading}
              autoComplete="new-password"
            />
          </div>

          <button type="submit" disabled={loading}>
            {loading ? 'Creating account...' : 'Sign up'}
          </button>
        </form>
        
        <div className="auth-footer">
          Already have an account? <Link to="/login">Sign in</Link>
        </div>
      </main>
    </div>
  );
}
