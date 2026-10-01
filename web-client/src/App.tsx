import { useState } from 'react';
import { BrowserRouter, Routes, Route, NavLink, Navigate, useNavigate } from 'react-router-dom';
import { LayoutDashboard, Receipt, MessageSquareWarning, BarChart3, LogOut, UploadCloud, Menu, X, User } from 'lucide-react';

import { Landing } from './components/Landing';
import { Login } from './components/Login';
import { Register } from './components/Register';
import { Dashboard } from './components/Dashboard';
import { Transactions } from './components/Transactions';
import { Clarifications } from './components/Clarifications';
import { Reports } from './components/Reports';
import { Profile } from './components/Profile';

import { Upload } from './components/Upload';

function Sidebar({ isOpen, setIsOpen }: { isOpen: boolean, setIsOpen: (val: boolean) => void }) {
  const navigate = useNavigate();
  const handleLogout = () => {
    localStorage.removeItem('token');
    navigate('/login');
  };

  return (
    <div className={`sidebar ${isOpen ? 'open' : ''}`}>
      <button className="mobile-close-btn" onClick={() => setIsOpen(false)}><X size={24} /></button>
      <div className="sidebar-logo">Ledger</div>
      
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        <NavLink to="/dashboard" className={({ isActive }) => `nav-link ${isActive ? 'active' : ''}`}>
          <LayoutDashboard size={18} /> Dashboard
        </NavLink>
        <NavLink to="/transactions" className={({ isActive }) => `nav-link ${isActive ? 'active' : ''}`}>
          <Receipt size={18} /> Transactions
        </NavLink>
        <NavLink to="/upload" className={({ isActive }) => `nav-link ${isActive ? 'active' : ''}`}>
          <UploadCloud size={18} /> Upload Receipt
        </NavLink>
        <NavLink to="/clarifications" className={({ isActive }) => `nav-link ${isActive ? 'active' : ''}`}>
          <MessageSquareWarning size={18} /> Questions
        </NavLink>
        <NavLink to="/reports" className={({ isActive }) => `nav-link ${isActive ? 'active' : ''}`}>
          <BarChart3 size={18} /> Reports
        </NavLink>
        <div style={{ margin: '16px 0', borderTop: '1px solid rgba(255,255,255,0.05)' }}></div>
        <NavLink to="/profile" className={({ isActive }) => `nav-link ${isActive ? 'active' : ''}`}>
          <User size={18} /> Profile
        </NavLink>
      </div>

      <button onClick={handleLogout} className="nav-link" style={{ background: 'transparent', border: 'none', color: 'var(--status-danger)', width: '100%', justifyContent: 'flex-start', boxShadow: 'none', paddingLeft: '8px' }}>
        <LogOut size={18} /> Log out
      </button>
    </div>
  );
}

function Layout({ children }: { children: React.ReactNode }) {
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);

  return (
    <div className="layout">
      <div className="landing-bg-grid" />
      <div className="landing-spotlight" />
      
      <div className="mobile-top-bar">
        <div className="sidebar-logo" style={{ margin: 0, padding: 0 }}>Ledger</div>
        <button className="mobile-menu-btn" onClick={() => setIsSidebarOpen(true)}>
          <Menu size={24} />
        </button>
      </div>

      <Sidebar isOpen={isSidebarOpen} setIsOpen={setIsSidebarOpen} />
      <main className="main-content">
        {children}
      </main>
    </div>
  );
}

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const token = localStorage.getItem('token');
  if (!token) return <Navigate to="/login" replace />;
  return <Layout>{children}</Layout>;
}

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Landing />} />
        
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        
        {/* Main App Routes */}
        <Route path="/dashboard" element={<PrivateRoute><Dashboard /></PrivateRoute>} />
        <Route path="/transactions" element={<PrivateRoute><Transactions /></PrivateRoute>} />
        <Route path="/upload" element={<PrivateRoute><Upload /></PrivateRoute>} />
        <Route path="/clarifications" element={<PrivateRoute><Clarifications /></PrivateRoute>} />
        <Route path="/reports" element={<PrivateRoute><Reports /></PrivateRoute>} />
        <Route path="/profile" element={<PrivateRoute><Profile /></PrivateRoute>} />
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
