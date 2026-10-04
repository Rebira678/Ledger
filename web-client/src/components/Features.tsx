import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Menu, X, ArrowLeft } from 'lucide-react';

export function Features() {
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
        
        <h1 style={{ fontSize: '2.5rem', marginBottom: '1rem', fontWeight: 600 }}>Ledger Features</h1>
        <p style={{ color: '#94a3b8', marginBottom: '2rem' }}>A comprehensive breakdown of Ledger's capabilities.</p>

        <section style={{ marginBottom: '2rem' }}>
          <h2 style={{ fontSize: '1.5rem', marginBottom: '1rem', color: '#e2e8f0' }}>Automated SMS Ingestion</h2>
          <p style={{ marginBottom: '1rem', color: '#cbd5e1' }}>
            The Ledger Android App runs silently in the background, intercepting incoming bank and mobile-money SMS notifications in real-time. Without any manual entry, it extracts amounts, merchant IDs, and timestamps instantly.
          </p>
        </section>

        <section style={{ marginBottom: '2rem' }}>
          <h2 style={{ fontSize: '1.5rem', marginBottom: '1rem', color: '#e2e8f0' }}>AI-Powered Categorization</h2>
          <p style={{ marginBottom: '1rem', color: '#cbd5e1' }}>
            Raw banking data is notoriously cryptic. Our local LLM processing engine converts confusing strings like "POS-98340-ACME" into highly accurate, human-readable categories (e.g. "Groceries - Acme Corp").
          </p>
        </section>

        <section style={{ marginBottom: '2rem' }}>
          <h2 style={{ fontSize: '1.5rem', marginBottom: '1rem', color: '#e2e8f0' }}>Clarification Engine</h2>
          <p style={{ marginBottom: '1rem', color: '#cbd5e1' }}>
            If a transaction is ambiguous, Ledger doesn't guess. It temporarily places the transaction in a "needs review" queue and asks you a natural-language clarifying question so the AI can learn your specific habits over time.
          </p>
        </section>

        <section style={{ marginBottom: '2rem' }}>
          <h2 style={{ fontSize: '1.5rem', marginBottom: '1rem', color: '#e2e8f0' }}>Weekly Narrative Reports</h2>
          <p style={{ marginBottom: '1rem', color: '#cbd5e1' }}>
            Get beyond just charts. Every week, the AI generates a customized, narrative report detailing your financial health, warning you of anomalous spending spikes, and celebrating when you stick to your budget.
          </p>
        </section>
      </main>
    </div>
  );
}
