import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Layers } from 'lucide-react';

export function ForgotPassword() {
  const [email, setEmail] = useState('');
  const [loading, setLoading] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');
    
    try {
      const res = await fetch('/v1/auth/forgot-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email })
      });
      
      const debugToken = res.headers.get('X-Debug-Reset-Token');
      
      if (!res.ok) {
        throw new Error("We couldn't process your request right now. Please try again later.");
      }
      
      if (debugToken) {
        console.log("DEBUG: Reset Token is:", debugToken);
        // Automatically redirect to reset page for local development / portfolio demonstration purposes
        window.location.href = `/reset-password?token=${debugToken}`;
        return;
      }
      
      setSuccess(true);
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
          <h1>Reset your password</h1>
          <p>Enter your email to receive a password reset link.</p>
        </div>
        
        {error && (
          <div className="error-banner">
            {error}
          </div>
        )}

        {success ? (
          <div style={{ textAlign: 'center', marginBottom: '24px' }}>
            <p style={{ color: 'var(--status-success)', marginBottom: '16px' }}>Check your email for a link to reset your password.</p>
            <Link to="/login" className="btn-primary" style={{ display: 'inline-block', width: '100%' }}>Back to Login</Link>
          </div>
        ) : (
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

            <button type="submit" disabled={loading}>
              {loading ? 'Sending link...' : 'Send reset link'}
            </button>
          </form>
        )}
        
        <div className="auth-footer">
          Remembered your password? <Link to="/login">Sign in</Link>
        </div>
      </main>
    </div>
  );
}
