import { useState } from 'react';
import { useNavigate, Link, useSearchParams } from 'react-router-dom';
import { Layers } from 'lucide-react';

export function ResetPassword() {
  const [searchParams] = useSearchParams();
  const [token, setToken] = useState(searchParams.get('token') || '');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState('');
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!token) {
      setError("Please enter the temporary password code sent to your email.");
      return;
    }
    
    setLoading(true);
    setError('');
    
    try {
      const res = await fetch('/v1/auth/reset-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token, password })
      });
      
      const data = await res.json().catch(() => ({}));
      
      if (!res.ok) {
        let errorMsg = data.error?.message || data.message || "We couldn't reset your password. Please try again.";
        if (errorMsg.includes('must be at least 8')) {
          errorMsg = "Your password must be at least 8 characters long.";
        } else if (errorMsg.includes('invalid or expired')) {
          errorMsg = "This reset link has expired or is invalid. Please request a new one.";
        }
        throw new Error(errorMsg);
      }
      
      setSuccess(true);
      setTimeout(() => {
        navigate('/login');
      }, 3000);
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
          <h1>Create new password</h1>
          <p>Please enter your new secure password.</p>
        </div>
        
        {error && (
          <div className="error-banner">
            {error}
          </div>
        )}

        {success ? (
          <div style={{ textAlign: 'center', marginBottom: '24px' }}>
            <p style={{ color: 'var(--status-success)', marginBottom: '16px' }}>Your password has been successfully reset! Redirecting to login...</p>
            <Link to="/login" className="btn-primary" style={{ display: 'inline-block', width: '100%' }}>Go to Login</Link>
          </div>
        ) : (
          <form onSubmit={handleSubmit}>
            <div className="form-group">
              <label>Temporary Password Code</label>
              <input 
                type="text" 
                placeholder="ABCDEF" 
                required 
                value={token} 
                onChange={e => setToken(e.target.value.toUpperCase())} 
                disabled={loading}
                autoComplete="off"
              />
            </div>

            <div className="form-group">
              <label>New Password</label>
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
              {loading ? 'Resetting...' : 'Change Password'}
            </button>
          </form>
        )}
      </main>
    </div>
  );
}
