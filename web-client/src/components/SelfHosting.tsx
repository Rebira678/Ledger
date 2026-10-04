import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Menu, X, ArrowLeft } from 'lucide-react';

export function SelfHosting() {
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
        
        <h1 style={{ fontSize: '2.5rem', marginBottom: '1rem', fontWeight: 600 }}>Self-Hosting Ledger</h1>
        <p style={{ color: '#94a3b8', marginBottom: '2rem' }}>Take ownership of your financial infrastructure.</p>

        <section style={{ marginBottom: '2rem' }}>
          <h2 style={{ fontSize: '1.5rem', marginBottom: '1rem', color: '#e2e8f0' }}>Why Self-Host?</h2>
          <p style={{ marginBottom: '1rem', color: '#cbd5e1' }}>
            When you run Ledger yourself, you completely eliminate the middleman. Your data flows directly from your Android phone to a server you control. Nobody can mine it, sell it, or accidentally expose it in a data breach.
          </p>
        </section>

        <section style={{ marginBottom: '2rem' }}>
          <h2 style={{ fontSize: '1.5rem', marginBottom: '1rem', color: '#e2e8f0' }}>Docker Deployment</h2>
          <p style={{ marginBottom: '1rem', color: '#cbd5e1' }}>
            We provide a production-ready `docker-compose.yml` that seamlessly spins up the PostgreSQL database and the Go server. All you need is Docker installed on your host machine.
          </p>
          <pre style={{ background: 'rgba(0,0,0,0.5)', padding: '16px', borderRadius: '8px', color: '#10b981', overflowX: 'auto', marginBottom: '1rem' }}>
            <code>
              cp .env.example .env{'\n'}
              # Edit your .env with a secure JWT signing key{'\n'}
              docker compose up --build -d
            </code>
          </pre>
        </section>

        <section style={{ marginBottom: '2rem' }}>
          <h2 style={{ fontSize: '1.5rem', marginBottom: '1rem', color: '#e2e8f0' }}>Pairing the Mobile App</h2>
          <p style={{ marginBottom: '1rem', color: '#cbd5e1' }}>
            Once your server is running, simply install the Ledger APK on your Android device. Under settings, update the "API Base URL" to point to your hosted instance, and log in with the account you registered on your web dashboard.
          </p>
        </section>
      </main>
    </div>
  );
}
