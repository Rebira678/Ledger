import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Menu, X, ArrowLeft } from 'lucide-react';

export function Community() {
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
        
        <h1 style={{ fontSize: '2.5rem', marginBottom: '1rem', fontWeight: 600 }}>Community</h1>
        <p style={{ color: '#94a3b8', marginBottom: '2rem' }}>Connect, contribute, and build with other sovereign individuals.</p>

        <section style={{ marginBottom: '2rem', padding: '24px', background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.05)', borderRadius: '12px' }}>
          <h2 style={{ fontSize: '1.5rem', marginBottom: '1rem', color: '#e2e8f0' }}>GitHub Discussions</h2>
          <p style={{ marginBottom: '1.5rem', color: '#cbd5e1' }}>
            The best place to ask questions, suggest features, and discuss architectural decisions is on our GitHub Discussions board.
          </p>
          <a href="https://github.com/Rebira678/Ledger" target="_blank" rel="noreferrer" className="btn-secondary" style={{ display: 'inline-flex', padding: '10px 20px', borderRadius: '8px' }}>
            Join the Discussion
          </a>
        </section>

        <section style={{ marginBottom: '2rem', padding: '24px', background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.05)', borderRadius: '12px' }}>
          <h2 style={{ fontSize: '1.5rem', marginBottom: '1rem', color: '#e2e8f0' }}>Contributing to Ledger</h2>
          <p style={{ marginBottom: '1rem', color: '#cbd5e1' }}>
            Ledger relies heavily on parser plugins to understand different bank SMS formats. If your local bank or mobile money provider isn't supported yet, contributing a parser plugin is the most impactful way to help. 
          </p>
          <p style={{ color: '#cbd5e1' }}>
            Check out the `internal/parsers` directory in the repository to see examples of existing plugins.
          </p>
        </section>
      </main>
    </div>
  );
}
