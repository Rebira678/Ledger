import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Smartphone, Zap, ArrowRight, Database, Key, ShieldCheck, LayoutDashboard, Receipt, UploadCloud, MessageSquareWarning, BarChart3, Menu, X } from 'lucide-react';

export function Landing() {
  const [isMenuOpen, setIsMenuOpen] = useState(false);

  return (
    <div className="landing-container">
      <div className="landing-bg-grid" />
      <div className="landing-spotlight" />

      {/* Navigation */}
      <nav className="landing-nav fade-in-down">
        <div className="landing-logo">
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

      {/* Hero Section */}
      <header className="hero-section">

        <div className="hero-pill animate-fade-in-up" style={{ animationDelay: '100ms' }}>
          <span className="pill-dot"></span>
          Powered by Local AI
        </div>

        <h1 className="hero-title animate-fade-in-up" style={{ animationDelay: '200ms' }}>
          Never ask "Where did<br />
          <span className="text-gradient-metallic">my money go?</span>" again.
        </h1>
        
        <p className="hero-subtitle animate-fade-in-up" style={{ animationDelay: '300ms' }}>
          You had $5,000 on Friday. By Monday, it's $1,000. Ledger intercepts your banking SMS notifications and automatically categorizes every expense so you instantly know exactly what drained your balance.<br/><br/>
          <span className="text-highlight">Total clarity. Zero manual entry.</span>
        </p>
        
        <div className="hero-actions animate-fade-in-up" style={{ animationDelay: '400ms' }}>
          <a href="https://github.com/Rebira678/Ledger/releases/latest/download/app-release.apk" download="Ledger.apk" className="btn-primary">
            <Smartphone size={18} style={{ marginRight: 4 }} />
            Download for Android
          </a>
          <Link to="/dashboard" className="btn-secondary">
            Open Web App <ArrowRight size={16} style={{ marginLeft: 4 }} />
          </Link>
        </div>
      </header>

      {/* Dashboard Preview Graphic */}
      <div className="hero-preview-wrapper animate-fade-in-up" style={{ animationDelay: '600ms' }}>
        <div className="hero-preview-glow"></div>

        {/* Floating Accents */}
        <div className="floating-card accent-1">
          <Zap size={20} color="#10b981" />
          <span>Categorized by AI</span>
        </div>
        <div className="floating-card accent-2">
          <ShieldCheck size={20} color="#3b82f6" />
          <span>100% Sovereign Data</span>
        </div>

        <div className="hero-preview">
          <div className="preview-header">
            <div className="preview-dots">
              <span /> <span /> <span />
            </div>
            <div className="preview-url">
              <ShieldCheck size={12} style={{ marginRight: 6, color: '#10b981' }} />
              ledger.local/dashboard
            </div>
          </div>
          <div className="preview-body">
            <div className="preview-sidebar">
              <div className="sidebar-group">Ledger App</div>
              <div className="sidebar-item active"><LayoutDashboard size={14} /> Dashboard</div>
              <div className="sidebar-item"><Receipt size={14} /> Transactions</div>
              <div className="sidebar-item"><UploadCloud size={14} /> Upload Receipt</div>
              <div className="sidebar-item"><MessageSquareWarning size={14} /> Questions</div>
              <div className="sidebar-item"><BarChart3 size={14} /> Reports</div>
            </div>
            <div className="preview-content">
              <div className="preview-metrics">
                <div className="metric-card">
                  <div className="metric-label">Total Spent (30d)</div>
                  <div className="metric-value">$4,285.00</div>
                  <div className="metric-trend up">+12% from last month</div>
                </div>
                <div className="metric-card">
                  <div className="metric-label">Categorized</div>
                  <div className="metric-value">98.2%</div>
                  <div className="metric-trend neutral">via local LLM</div>
                </div>
                <div className="metric-card">
                  <div className="metric-label">Pending Clarifications</div>
                  <div className="metric-value">2</div>
                  <div className="metric-trend down">Action required</div>
                </div>
              </div>

              <div className="preview-charts-row">
                <div className="preview-chart-main">
                  <div className="chart-header">Spend over time</div>
                  <div className="chart-bars">
                    {[40, 70, 45, 90, 65, 85, 30, 50, 100, 75, 40, 60].map((h, i) => (
                      <div key={i} className="chart-bar-wrapper">
                        <div className="chart-bar" style={{ height: `${h}%` }}>
                          <div className="chart-bar-glow"></div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
                <div className="preview-recent">
                  <div className="chart-header">Recent Transactions</div>
                  <div className="recent-list">
                    <div className="recent-item">
                      <div className="r-icon netflix">N</div>
                      <div className="r-details">
                        <div className="r-title">Netflix</div>
                        <div className="r-category">Entertainment</div>
                      </div>
                      <div className="r-amount">-$15.99</div>
                    </div>
                    <div className="recent-item">
                      <div className="r-icon uber">U</div>
                      <div className="r-details">
                        <div className="r-title">Uber Eats</div>
                        <div className="r-category">Food</div>
                      </div>
                      <div className="r-amount">-$32.50</div>
                    </div>
                    <div className="recent-item">
                      <div className="r-icon salary">S</div>
                      <div className="r-details">
                        <div className="r-title">Acme Corp</div>
                        <div className="r-category">Salary</div>
                      </div>
                      <div className="r-amount positive">+$4,200.00</div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Tech Stack Marquee / Trust */}
      <div className="trust-section">
        <p className="trust-label">Seamlessly processing SMS alerts from Ethiopian Institutions</p>
        <div className="trust-logos-marquee">
          <div className="marquee-content">
             <span>Commercial Bank of Ethiopia</span><span className="dot">•</span>
             <span>Telebirr</span><span className="dot">•</span>
             <span>Dashen Bank</span><span className="dot">•</span>
             <span>Awash Bank</span><span className="dot">•</span>
             <span>Bank of Abyssinia</span><span className="dot">•</span>
             <span>Zemen Bank</span><span className="dot">•</span>
             <span>Hibret Bank</span><span className="dot">•</span>
             <span>Cooperative Bank of Oromia</span><span className="dot">•</span>
          </div>
          <div className="marquee-content" aria-hidden="true">
             <span>Commercial Bank of Ethiopia</span><span className="dot">•</span>
             <span>Telebirr</span><span className="dot">•</span>
             <span>Dashen Bank</span><span className="dot">•</span>
             <span>Awash Bank</span><span className="dot">•</span>
             <span>Bank of Abyssinia</span><span className="dot">•</span>
             <span>Zemen Bank</span><span className="dot">•</span>
             <span>Hibret Bank</span><span className="dot">•</span>
             <span>Cooperative Bank of Oromia</span><span className="dot">•</span>
          </div>
        </div>
      </div>

      {/* How It Works Section */}
      <section className="how-it-works-section">
        <div className="section-header">
          <h2 className="reveal-text">How Ledger Works</h2>
          <p>Automate your expense tracking securely using your mobile device.</p>
        </div>
        
        <div className="pipeline-container">
           <div className="pipeline-step">
              <div className="step-icon-wrapper"><Smartphone size={32} /></div>
              <h4 className="step-title">1. Download the App</h4>
              <p className="step-desc">First, you must download the Android app. If you choose not to use the app, the only way to track your expenses is by uploading your transactions manually via the web.</p>
           </div>
           
           <div className="pipeline-connector">
              <div className="connector-line"></div>
              <div className="connector-dot"></div>
           </div>

           <div className="pipeline-step">
              <div className="step-icon-wrapper"><MessageSquareWarning size={32} /></div>
              <h4 className="step-title">2. Grant Notification Access</h4>
              <p className="step-desc">After downloading, give the app permission to read your notifications. Ledger intelligently filters notifications to only track messages coming directly from official banks.</p>
           </div>
           
           <div className="pipeline-connector">
              <div className="connector-line"></div>
              <div className="connector-dot"></div>
           </div>

           <div className="pipeline-step">
              <div className="step-icon-wrapper"><ShieldCheck size={32} /></div>
              <h4 className="step-title">3. 100% Secure & Read-Only</h4>
              <p className="step-desc">Ledger is completely secure. It does not touch your money and cannot make transfers. It is strictly a read-only tool designed only to extract and track your transaction history.</p>
           </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="features-section">
        <div className="section-header">
          <h2 className="reveal-text">Know your spending. Keep your secrets.</h2>
          <p>You shouldn't have to hand over your bank credentials to a third-party app just to understand your own spending habits.</p>
        </div>
        <div className="bento-grid">
          <div className="bento-card col-span-2 bento-hover">
            <div className="bento-glow"></div>
            <div className="bento-content">
              <div className="bento-image-wrapper">
                <img src="/assets/senior_3d_sms_phone.png" alt="Android SMS Phone 3D" className="bento-image" />
              </div>
              <div className="feature-icon"><Smartphone size={24} /></div>
              <h3>Full Android Experience</h3>
              <p>The Android app isn't just a background agent—it's a complete financial dashboard in your pocket. It securely captures incoming bank SMS alerts and visualizes your spending without ever needing to open the web.</p>
            </div>
          </div>
          <div className="bento-card bento-hover">
            <div className="bento-glow"></div>
            <div className="bento-content">
              <div className="bento-image-wrapper">
                <img src="/assets/senior_3d_ai_chip.png" alt="AI Neural Core 3D" className="bento-image" />
              </div>
              <div className="feature-icon"><Zap size={24} /></div>
              <h3>It Actually Understands Your Receipts</h3>
              <p>Ever seen a charge for "POS-19348-ACME" and wondered what you bought? Our local AI translates confusing bank jargon into plain English categories like "Groceries" or "Uber".</p>
            </div>
          </div>
          <div className="bento-card bento-hover">
            <div className="bento-glow"></div>
            <div className="bento-content">
              <div className="bento-image-wrapper">
                <img src="/assets/senior_3d_database.png" alt="Database Vault 3D" className="bento-image" />
              </div>
              <div className="feature-icon"><Database size={24} /></div>
              <h3>Never Double-Counts a Penny</h3>
              <p>We use bank-grade database architecture to ensure that if a transaction is delayed or an SMS is sent twice, your balance is never miscalculated. Total accuracy, always.</p>
            </div>
          </div>
          <div className="bento-card col-span-2 bento-hover">
            <div className="bento-glow"></div>
            <div className="bento-content">
              <div className="bento-image-wrapper">
                <img src="/assets/senior_3d_lock.png" alt="Privacy Lock 3D" className="bento-image" />
              </div>
              <div className="feature-icon"><Key size={24} /></div>
              <h3>Your Money. Your Data. Nobody Else's.</h3>
              <p>Big finance apps sell your spending habits to advertisers. Ledger is 100% open-source and runs on your own hardware. We literally cannot see your data.</p>
            </div>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="cta-section">
        <div className="cta-glow-orb"></div>
        <div className="cta-content-wrapper">
          <h2 className="cta-title">Ready to finally see where your money goes?</h2>
          <p className="cta-subtitle">Join the movement of users who have stopped guessing and started knowing. Host your own Ledger instance today.</p>
          <div className="cta-buttons">
            <Link to="/register" className="btn-primary-large">
              Deploy Ledger Now <ArrowRight size={20} />
            </Link>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="landing-footer">
        <div className="footer-content">
          <div className="footer-brand">
            <div className="landing-logo" style={{ color: '#fff' }}>
              <div className="logo-box" style={{ background: '#fff', color: '#000' }}>L</div>
              Ledger
            </div>
            <p className="footer-description">
              The open-source, AI-powered financial agent for sovereign individuals.
            </p>
            <div className="footer-socials">
              <a href="https://x.com/Rebira678" target="_blank" rel="noreferrer">X / Twitter</a>
              <a href="https://linkedin.com/in/rebira" target="_blank" rel="noreferrer">LinkedIn</a>
              <a href="https://github.com/Rebira678/Ledger" target="_blank" rel="noreferrer">GitHub</a>
            </div>
          </div>
          <div className="footer-grid">
            <div className="footer-column">
              <h4>Product</h4>
              <Link to="/features">Features</Link>
              <Link to="/security">Security</Link>
              <Link to="/self-hosting">Self-Hosting</Link>
              <Link to="/changelog">Changelog</Link>
            </div>
            <div className="footer-column">
              <h4>Resources</h4>
              <a href="https://github.com/Rebira678/Ledger/blob/main/Ledger_API_Contract.pdf" target="_blank" rel="noreferrer">API Reference</a>
              <Link to="/community">Community</Link>
              <a href="https://github.com/Rebira678/Ledger" target="_blank" rel="noreferrer">GitHub</a>
            </div>
            <div className="footer-column">
              <h4>Legal</h4>
              <Link to="/privacy">Privacy Policy</Link>
              <Link to="/terms">Terms of Service</Link>
              <Link to="/license">License</Link>
            </div>
          </div>
        </div>
        <div className="footer-bottom-bar">
          &copy; {new Date().getFullYear()} Ledger Inc. All rights reserved.
        </div>
      </footer>
    </div>
  );
}
