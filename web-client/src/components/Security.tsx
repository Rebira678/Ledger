import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Menu, X, ArrowLeft } from 'lucide-react';

export function Security() {
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const navigate = useNavigate();

  return (
    <div className="landing-container">
      <div className="landing-bg-grid" />
      <div className="landing-spotlight" />

      {/* Navigation */}
      <nav className="landing-nav fade-in-down">
        <div className="landing-logo" onClick={() => navigate('/')} style={{ cursor: 'pointer' }}>
          <div className="logo-box">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: 14, height: 14 }}>
              <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" />
            </svg>
          </div>
          Ledger
        </div>
        <button className="mobile-menu-btn" onClick={() => setIsMenuOpen(!isMenuOpen)}>
          {isMenuOpen ? <X size={24} /> : <Menu size={24} />}
        </button>
        <div className={`landing-nav-links ${isMenuOpen ? 'mobile-open' : ''}`}>
          <Link to="/login" className="nav-link-ghost">Log in</Link>
          <Link to="/register" className="nav-link-primary">Get Started</Link>
        </div>
      </nav>

      <main style={{ maxWidth: 800, margin: '120px auto 60px', padding: '0 24px', color: '#fff', lineHeight: 1.6 }}>
        <button onClick={() => navigate(-1)} style={{ background: 'none', border: 'none', color: '#94a3b8', display: 'flex', alignItems: 'center', cursor: 'pointer', marginBottom: 24, fontSize: '0.9rem' }}>
          <ArrowLeft size={16} style={{ marginRight: 6 }} /> Back
        </button>
        
        <h1 style={{ fontSize: '2.5rem', marginBottom: '1rem', fontWeight: 600 }}>Security & Governance</h1>
        <p style={{ color: '#94a3b8', marginBottom: '2rem' }}>How we protect your most sensitive data.</p>

        <section style={{ marginBottom: '2rem' }}>
          <h2 style={{ fontSize: '1.5rem', marginBottom: '1rem', color: '#e2e8f0' }}>End-to-End Encryption</h2>
          <p style={{ marginBottom: '1rem', color: '#cbd5e1' }}>
            Data in transit between the Ledger Android app and the Ledger Backend is strictly secured via HTTPS. The API uses stateless JWT authentication, ensuring that only paired devices and authenticated web clients can communicate with your server.
          </p>
        </section>

        <section style={{ marginBottom: '2rem' }}>
          <h2 style={{ fontSize: '1.5rem', marginBottom: '1rem', color: '#e2e8f0' }}>No Plaid, No Bank Logins</h2>
          <p style={{ marginBottom: '1rem', color: '#cbd5e1' }}>
            Ledger fundamentally rejects the premise of sharing bank credentials. Because we operate purely on your incoming SMS notifications, we never ask for your banking usernames, passwords, or OAuth tokens. If our system is breached, the attacker cannot drain your accounts.
          </p>
        </section>

        <section style={{ marginBottom: '2rem' }}>
          <h2 style={{ fontSize: '1.5rem', marginBottom: '1rem', color: '#e2e8f0' }}>Total Infrastructure Control</h2>
          <p style={{ marginBottom: '1rem', color: '#cbd5e1' }}>
            The truest form of security is physical and digital ownership. Ledger's open-source architecture enables you to host the entire stack (Postgres Database, Go Backend, React Frontend) on your own hardware or VPC.
          </p>
        </section>
      </main>
    </div>
  );
}
