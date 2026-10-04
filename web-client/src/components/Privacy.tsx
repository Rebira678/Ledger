import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Menu, X, ArrowLeft } from 'lucide-react';

export function Privacy() {
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
        
        <h1 style={{ fontSize: '2.5rem', marginBottom: '1rem', fontWeight: 600 }}>Privacy Policy</h1>
        <p style={{ color: '#94a3b8', marginBottom: '2rem' }}>Last updated: October 2026</p>

        <section style={{ marginBottom: '2rem' }}>
          <h2 style={{ fontSize: '1.5rem', marginBottom: '1rem', color: '#e2e8f0' }}>1. Sovereign Data Commitment</h2>
          <p style={{ marginBottom: '1rem', color: '#cbd5e1' }}>
            Ledger is designed for sovereign individuals. Our core architecture ensures that we literally cannot see your financial data. All SMS interception, parsing, and categorization happens locally on your edge devices (Android) or your self-hosted server environment.
          </p>
        </section>

        <section style={{ marginBottom: '2rem' }}>
          <h2 style={{ fontSize: '1.5rem', marginBottom: '1rem', color: '#e2e8f0' }}>2. Data We Do Not Collect</h2>
          <ul style={{ listStyleType: 'disc', paddingLeft: '1.5rem', color: '#cbd5e1', marginBottom: '1rem' }}>
            <li style={{ marginBottom: '0.5rem' }}>We do not collect your transaction history.</li>
            <li style={{ marginBottom: '0.5rem' }}>We do not collect your bank account numbers or balances.</li>
            <li style={{ marginBottom: '0.5rem' }}>We do not sell your spending habits to third-party advertisers.</li>
          </ul>
        </section>

        <section style={{ marginBottom: '2rem' }}>
          <h2 style={{ fontSize: '1.5rem', marginBottom: '1rem', color: '#e2e8f0' }}>3. Self-Hosted Telemetry (Optional)</h2>
          <p style={{ marginBottom: '1rem', color: '#cbd5e1' }}>
            If you self-host Ledger, the system generates internal crash reports and error logs. These logs never leave your server unless you explicitly configure them to be sent to a third-party observability platform (like Grafana or Datadog). 
          </p>
        </section>

        <section style={{ marginBottom: '2rem' }}>
          <h2 style={{ fontSize: '1.5rem', marginBottom: '1rem', color: '#e2e8f0' }}>4. Local AI Processing</h2>
          <p style={{ marginBottom: '1rem', color: '#cbd5e1' }}>
            Any financial categorization performed by AI happens either directly via a local LLM or securely through an API endpoint of your choosing. You control the API keys and the data flow.
          </p>
        </section>
      </main>

    </div>
  );
}
