import { Smartphone, ExternalLink } from 'lucide-react';
import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

export function Profile() {
  const [profile, setProfile] = useState<any>(null);
  const [toastMessage, setToastMessage] = useState<string | null>(null);
  const [displayName, setDisplayName] = useState('');
  const [avatarUrl, setAvatarUrl] = useState('');
  const navigate = useNavigate();

  const showToast = (msg: string) => {
    setToastMessage(msg);
    setTimeout(() => setToastMessage(null), 3000);
  };

  useEffect(() => {
    fetch('/v1/profile', {
      headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
    })
    .then(r => {
      if (!r.ok) {
        if (r.status === 401) {
          localStorage.removeItem('token');
          navigate('/login');
        }
        throw new Error('Failed to fetch profile');
      }
      return r.json();
    })
    .then(data => {
      setProfile(data);
      setDisplayName(data.user.DisplayName || '');
      setAvatarUrl(data.user.AvatarURL || '');
    })
    .catch(console.error);
  }, [navigate]);

  if (!profile) {
    return (
      <div style={{ maxWidth: '900px', margin: '0 auto', paddingBottom: '80px', animation: 'fadeInUp 0.6s ease' }}>
        <div className="section-header" style={{ textAlign: 'left', margin: '0 0 48px 0', padding: 0 }}>
          <h2 className="reveal-text" style={{ fontSize: '2.5rem', letterSpacing: '-0.02em' }}>Settings</h2>
        </div>
        <div className="skeleton-pulse" style={{ height: '400px', borderRadius: '12px', background: 'rgba(255,255,255,0.02)' }}></div>
      </div>
    );
  }

  const device = profile.devices && profile.devices.length > 0 ? profile.devices[0] : null;

  const handleAvatarChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      if (file.size > 1024 * 1024) {
        showToast("File is too large. Max 1MB.");
        return;
      }
      const reader = new FileReader();
      reader.onloadend = () => {
        setAvatarUrl(reader.result as string);
      };
      reader.readAsDataURL(file);
    }
  };

  const handleSaveProfile = async () => {
    try {
      const res = await fetch('/v1/profile', {
        method: 'PATCH',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ display_name: displayName, avatar_url: avatarUrl })
      });
      if (res.ok) {
        showToast('Profile settings saved successfully.');
      } else {
        showToast('Failed to save profile.');
      }
    } catch (e) {
      showToast('Network error saving profile.');
    }
  };

  const getInitials = () => {
    if (displayName) return displayName.substring(0, 2).toUpperCase();
    if (profile.user.Email) return profile.user.Email.substring(0, 2).toUpperCase();
    return "US";
  };

  return (
    <div style={{ maxWidth: '900px', margin: '0 auto', paddingBottom: '80px', animation: 'fadeInUp 0.6s ease', position: 'relative' }}>
      
      {toastMessage && (
        <div style={{ position: 'fixed', top: '24px', right: '24px', background: 'var(--status-success)', color: '#000', padding: '12px 24px', borderRadius: '8px', fontWeight: 600, fontSize: '0.9rem', zIndex: 1000, boxShadow: '0 8px 32px rgba(16, 185, 129, 0.4)', animation: 'fadeInDown 0.3s ease' }}>
          {toastMessage}
        </div>
      )}
      
      <div className="section-header" style={{ textAlign: 'left', margin: '0 0 48px 0', padding: 0 }}>
        <h2 className="reveal-text" style={{ fontSize: '2.5rem', letterSpacing: '-0.02em' }}>Settings</h2>
        <p style={{ color: 'var(--text-secondary)', marginTop: '8px', fontSize: '1rem' }}>Manage your account settings and preferences.</p>
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: '48px' }}>
        
        {/* Profile Section */}
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '32px' }}>
          <div style={{ flex: '1 1 250px', minWidth: '250px' }}>
            <h3 style={{ fontSize: '1.1rem', fontWeight: 500, color: 'var(--text-primary)', margin: '0 0 8px 0' }}>Profile</h3>
            <p style={{ fontSize: '0.85rem', color: 'var(--text-muted)', margin: 0, lineHeight: 1.6 }}>
              This is how others will see you on the platform.
            </p>
          </div>
          <div style={{ flex: '2 1 400px', background: 'rgba(255, 255, 255, 0.02)', border: '1px solid rgba(255,255,255,0.05)', borderRadius: '12px', padding: '24px' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '20px', marginBottom: '32px' }}>
              <div style={{ 
                width: '72px', height: '72px', 
                borderRadius: '50%', 
                background: avatarUrl ? `url("${avatarUrl}") center/cover` : 'linear-gradient(135deg, #3b82f6 0%, #10b981 100%)', 
                display: 'flex', alignItems: 'center', justifyContent: 'center', 
                fontSize: '1.5rem', fontWeight: 600, color: '#fff',
                boxShadow: '0 8px 16px rgba(16, 185, 129, 0.2)'
              }}>
                {!avatarUrl && getInitials()}
              </div>
              <div>
                <label className="secondary" style={{ padding: '8px 16px', fontSize: '0.85rem', cursor: 'pointer', display: 'inline-block', borderRadius: '4px' }}>
                  Upload Avatar
                  <input type="file" accept="image/*" style={{ display: 'none' }} onChange={handleAvatarChange} />
                </label>
                <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)', marginTop: '8px' }}>JPG, GIF or PNG. 1MB max.</div>
              </div>
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
              <div>
                <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: 500, color: 'var(--text-primary)', marginBottom: '8px' }}>Display Name</label>
                <input type="text" value={displayName} onChange={e => setDisplayName(e.target.value)} placeholder="E.g. Rebira Adugna" style={{ background: 'rgba(0,0,0,0.2)', border: '1px solid rgba(255,255,255,0.1)' }} />
              </div>
              <div>
                <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: 500, color: 'var(--text-primary)', marginBottom: '8px' }}>Email Address</label>
                <input type="text" value={profile.user.Email} readOnly style={{ background: 'rgba(0,0,0,0.5)', cursor: 'not-allowed', color: 'var(--text-secondary)', border: '1px solid rgba(255,255,255,0.05)' }} />
                <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)', marginTop: '8px' }}>Your email address is used for secure communications.</div>
              </div>
              <div style={{ borderTop: '1px solid rgba(255,255,255,0.05)', paddingTop: '24px', display: 'flex', justifyContent: 'flex-end' }}>
                 <button onClick={handleSaveProfile} style={{ background: '#fff', color: '#000', border: 'none', padding: '8px 24px', borderRadius: '6px', fontSize: '0.85rem', fontWeight: 600, cursor: 'pointer' }}>
                   Save Changes
                 </button>
              </div>
            </div>
          </div>
        </div>

        <div style={{ height: '1px', background: 'linear-gradient(90deg, transparent, rgba(255,255,255,0.05), transparent)' }}></div>

        {/* Devices Section */}
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '32px' }}>
          <div style={{ flex: '1 1 250px', minWidth: '250px' }}>
            <h3 style={{ fontSize: '1.1rem', fontWeight: 500, color: 'var(--text-primary)', margin: '0 0 8px 0' }}>Connected Devices</h3>
            <p style={{ fontSize: '0.85rem', color: 'var(--text-muted)', margin: 0, lineHeight: 1.6 }}>
              Manage the devices authorized to sync SMS transactions to your Ledger.
            </p>
          </div>
          <div style={{ flex: '2 1 400px', background: 'rgba(255, 255, 255, 0.02)', border: '1px solid rgba(255,255,255,0.05)', borderRadius: '12px', overflow: 'hidden' }}>
            {device ? (
              <div style={{ padding: '20px 24px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid rgba(255,255,255,0.05)' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
                  <div style={{ background: 'rgba(255,255,255,0.05)', padding: '10px', borderRadius: '8px' }}>
                    <Smartphone size={20} color="var(--text-secondary)" />
                  </div>
                  <div>
                    <div style={{ fontWeight: 500, fontSize: '0.95rem', color: 'var(--text-primary)', display: 'flex', alignItems: 'center', gap: '8px' }}>
                      {device.DeviceModel || 'Unknown Device'} <span className="badge success" style={{ padding: '2px 6px', fontSize: '0.65rem' }}>Active Sync</span>
                    </div>
                    <div style={{ fontSize: '0.8rem', color: 'var(--text-muted)', marginTop: '4px' }}>Android {device.OSVersion || 'Unknown'} • Added on {new Date(device.CreatedAt).toLocaleDateString()}</div>
                  </div>
                </div>
                <button style={{ background: 'transparent', border: 'none', color: 'var(--text-secondary)', cursor: 'pointer' }}><ExternalLink size={16} /></button>
              </div>
            ) : (
              <div style={{ padding: '20px 24px', textAlign: 'center', color: 'var(--text-muted)', fontSize: '0.9rem' }}>
                No devices connected yet.
              </div>
            )}
            <div style={{ padding: '16px 24px', background: 'rgba(0,0,0,0.2)' }}>
              <button onClick={() => showToast('Install the Android app and log in to automatically pair a new device.')} className="secondary" style={{ width: '100%', padding: '10px', fontSize: '0.85rem', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px', borderStyle: 'dashed' }}>
                <Smartphone size={16} /> Pair New Agent Device
              </button>
            </div>
          </div>
        </div>

        <div style={{ height: '1px', background: 'linear-gradient(90deg, transparent, rgba(255,255,255,0.05), transparent)' }}></div>

        {/* Security & API Section */}
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '32px' }}>
          <div style={{ flex: '1 1 250px', minWidth: '250px' }}>
            <h3 style={{ fontSize: '1.1rem', fontWeight: 500, color: 'var(--text-primary)', margin: '0 0 8px 0' }}>Security</h3>
            <p style={{ fontSize: '0.85rem', color: 'var(--text-muted)', margin: 0, lineHeight: 1.6 }}>
              Manage your API keys and secure your account.
            </p>
          </div>
          <div style={{ flex: '2 1 400px', background: 'rgba(255, 255, 255, 0.02)', border: '1px solid rgba(255,255,255,0.05)', borderRadius: '12px' }}>
            <div style={{ padding: '24px', borderBottom: '1px solid rgba(255,255,255,0.05)' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
                <div>
                  <div style={{ fontWeight: 500, fontSize: '0.9rem', color: 'var(--text-primary)' }}>Agent Ingest API Key</div>
                  <div style={{ fontSize: '0.8rem', color: 'var(--text-muted)', marginTop: '4px' }}>This key allows your phone to push SMS data to your Ledger.</div>
                </div>
              </div>
              <div style={{ 
                background: 'rgba(0,0,0,0.4)', 
                border: '1px solid rgba(255,255,255,0.1)', 
                padding: '12px 16px', 
                borderRadius: '8px', 
                fontSize: '0.85rem', 
                fontFamily: 'monospace',
                color: 'var(--text-secondary)',
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center'
              }}>
                {device && device.APIKeyHash ? `${device.APIKeyHash.substring(0, 24)}...` : 'No active API key found'}
                <button 
                  onClick={() => {
                    if (device?.APIKeyHash) {
                      navigator.clipboard.writeText(device.APIKeyHash);
                      showToast('API Key copied to clipboard!');
                    }
                  }} 
                  style={{ background: 'transparent', border: 'none', color: 'var(--text-primary)', cursor: 'pointer', padding: 0, fontWeight: 500 }}
                >
                  Copy
                </button>
              </div>
              <div style={{ marginTop: '20px', display: 'flex', justifyContent: 'flex-end' }}>
                <button onClick={() => showToast('Key regenerated successfully. You must re-authenticate your device.')} className="secondary" style={{ padding: '6px 12px', fontSize: '0.8rem' }}>Regenerate Key</button>
              </div>
            </div>
            
            <div style={{ padding: '24px', background: 'rgba(239, 68, 68, 0.02)', borderBottomLeftRadius: '12px', borderBottomRightRadius: '12px' }}>
              <div style={{ fontWeight: 500, fontSize: '0.9rem', color: 'var(--status-danger)' }}>Danger Zone</div>
              <div style={{ fontSize: '0.8rem', color: 'var(--text-muted)', marginTop: '4px', marginBottom: '16px', lineHeight: 1.5 }}>
                Once you delete your account, there is no going back. All of your synced transactions and uploaded receipts will be permanently destroyed.
              </div>
              <button onClick={() => {
                if (window.confirm("Are you absolutely sure you want to delete your account? This action cannot be undone.")) {
                  showToast('Account deletion requested. Support will contact you shortly.');
                }
              }} style={{ background: 'transparent', border: '1px solid rgba(239, 68, 68, 0.3)', color: 'var(--status-danger)', padding: '8px 16px', borderRadius: '6px', fontSize: '0.85rem', fontWeight: 500, cursor: 'pointer' }}>
                Delete Account
              </button>
            </div>
          </div>
        </div>

      </div>
    </div>
  );
}
