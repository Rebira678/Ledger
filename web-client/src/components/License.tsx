import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Menu, X, ArrowLeft } from 'lucide-react';

export function License() {
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
        
        <h1 style={{ fontSize: '2.5rem', marginBottom: '1rem', fontWeight: 600 }}>MIT License</h1>
        <p style={{ color: '#94a3b8', marginBottom: '2rem' }}>Copyright (c) 2026 Ledger</p>

        <section style={{ marginBottom: '2rem' }}>
          <p style={{ marginBottom: '1rem', color: '#cbd5e1' }}>
            Permission is hereby granted, free of charge, to any person obtaining a copy
            of this software and associated documentation files (the "Software"), to deal
            in the Software without restriction, including without limitation the rights
            to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
            copies of the Software, and to permit persons to whom the Software is
            furnished to do so, subject to the following conditions:
          </p>
          <p style={{ marginBottom: '1rem', color: '#cbd5e1' }}>
            The above copyright notice and this permission notice shall be included in all
            copies or substantial portions of the Software.
          </p>
          <p style={{ marginBottom: '1rem', color: '#cbd5e1', textTransform: 'uppercase' }}>
            The Software is provided "as is", without warranty of any kind, express or
            implied, including but not limited to the warranties of merchantability,
            fitness for a particular purpose and noninfringement. In no event shall the
            authors or copyright holders be liable for any claim, damages or other
            liability, whether in an action of contract, tort or otherwise, arising from,
            out of or in connection with the software or the use or other dealings in the
            Software.
          </p>
        </section>
      </main>

    </div>
  );
}
