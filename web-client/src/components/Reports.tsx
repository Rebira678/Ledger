import { useState, useEffect } from 'react';
import { ArrowRight, FileText } from 'lucide-react';

export function Reports() {
  const [reports, setReports] = useState<any[]>([]);

  useEffect(() => {
    async function fetchReports() {
      try {
        const res = await fetch('/v1/reports/weekly', {
          headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
        });
        if (res.ok) {
          const json = await res.json();
          const mapped = (json.reports || []).map((r: any) => ({
            id: r.report_id,
            period: r.period_name || 'Weekly Report',
            status: r.status || 'available'
          }));
          setReports(mapped);
        }
      } catch (err) {
        console.error(err);
      }
    }
    fetchReports();
  }, []);

  return (
    <div>
      <h1>Weekly Reports</h1>
      
      <div style={{ display: 'grid', gap: '16px' }}>
        {reports.length > 0 ? reports.map(r => (
          <div key={r.id} className="glass-card" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '24px' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
              <div style={{ background: 'var(--bg-canvas)', padding: '12px', borderRadius: '12px', color: 'var(--text-muted)', border: '1px solid var(--border-subtle)' }}>
                <FileText size={24} />
              </div>
              <div>
                <div style={{ fontSize: '1rem', fontWeight: 600, marginBottom: '4px', color: 'var(--text-primary)' }}>{r.period}</div>
                <div style={{ color: 'var(--text-muted)', fontSize: '0.875rem', display: 'flex', alignItems: 'center', gap: '8px' }}>
                  Status: <span className="badge success">{r.status}</span>
                </div>
              </div>
            </div>
            <button className="secondary">
              View Report <ArrowRight size={16} />
            </button>
          </div>
        )) : (
          <div className="glass-card" style={{ textAlign: 'center', padding: '48px 24px', color: 'var(--text-muted)' }}>
            <div style={{ fontSize: '1rem', marginBottom: '8px', fontWeight: 500, color: 'var(--text-primary)' }}>No reports yet</div>
            <div style={{ fontSize: '0.875rem' }}>The first one will be generated at the end of the week.</div>
          </div>
        )}
      </div>
    </div>
  );
}
