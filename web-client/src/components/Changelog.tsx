import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Menu, X, ArrowLeft } from 'lucide-react';

export function Changelog() {
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
        
        <h1 style={{ fontSize: '2.5rem', marginBottom: '1rem', fontWeight: 600 }}>Changelog</h1>
        <p style={{ color: '#94a3b8', marginBottom: '2rem' }}>All notable changes to Ledger will be documented here.</p>

        <section style={{ marginBottom: '2rem' }}>
          <h2 style={{ fontSize: '1.5rem', marginBottom: '0.5rem', color: '#e2e8f0' }}>v1.0.0 - Project Launch</h2>
          <span style={{ display: 'inline-block', background: '#10b981', color: '#000', padding: '2px 8px', borderRadius: '4px', fontSize: '0.8rem', fontWeight: 'bold', marginBottom: '1rem' }}>Initial Release</span>
          <ul style={{ listStyleType: 'disc', paddingLeft: '1.5rem', color: '#cbd5e1' }}>
            <li style={{ marginBottom: '0.5rem' }}>Core architecture finalized: Go backend, React frontend, Android client.</li>
            <li style={{ marginBottom: '0.5rem' }}>Added regex parsing plugins for Commercial Bank of Ethiopia and Telebirr.</li>
            <li style={{ marginBottom: '0.5rem' }}>Implemented local AI categorization fallback and clarification engine.</li>
            <li style={{ marginBottom: '0.5rem' }}>Built interactive Web Dashboard with complete bento-grid aesthetic.</li>
            <li style={{ marginBottom: '0.5rem' }}>Released complete documentation, deployment scripts, and docker-compose configurations.</li>
          </ul>
        </section>
      </main>
    </div>
  );
}
