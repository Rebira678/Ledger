import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid, Cell } from 'recharts';
import { ArrowUpRight, ArrowDownRight, AlertCircle } from 'lucide-react';

export function Dashboard() {
  const navigate = useNavigate();
  const [data, setData] = useState<any>(null);

  useEffect(() => {
    async function fetchDashboard() {
      try {
        const res = await fetch('/v1/dashboard/summary', {
          headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
        });
        if (res.ok) {
          const json = await res.json();
          setData(json);
        }
      } catch (err) {
        console.error(err);
      }
    }
    fetchDashboard();
  }, []);

  const chartData = data ? [
    { name: 'Income (+)', value: data.this_week_income || 0, fill: '#10b981' }, // Green
    { name: 'Expense (-)', value: data.this_week_spend || 0, fill: '#ef4444' }  // Red
  ] : [];

  const maxVal = Math.max(data?.this_week_income || 0, data?.this_week_spend || 0);
  const maxTick = Math.max(2000, Math.ceil(maxVal / 500) * 500);
  const yTicks = [];
  for (let i = 0; i <= maxTick; i += 500) {
    yTicks.push(i);
  }

  return (
    <div>
      {data && data.pending_clarifications !== undefined && (
        <div style={{
          background: data.pending_clarifications > 0 ? 'var(--status-warning-bg)' : 'rgba(16, 185, 129, 0.1)',
          border: `1px solid ${data.pending_clarifications > 0 ? 'rgba(245, 158, 11, 0.3)' : 'rgba(16, 185, 129, 0.3)'}`,
          borderRadius: '8px',
          padding: '12px 16px',
          marginBottom: '24px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          color: data.pending_clarifications > 0 ? 'var(--status-warning)' : 'var(--status-success)',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '0.875rem', fontWeight: 500 }}>
            <AlertCircle size={18} />
            <span>Needs Review: {data.pending_clarifications} transaction{data.pending_clarifications === 1 ? '' : 's'}</span>
          </div>
          <button onClick={() => navigate('/clarifications')} style={{ 
            background: data.pending_clarifications > 0 ? 'var(--status-warning)' : 'var(--status-success)', 
            color: '#000', 
            border: 'none', 
            padding: '6px 16px', 
            borderRadius: '6px', 
            fontSize: '0.85rem', 
            fontWeight: 600,
            cursor: 'pointer',
            width: 'auto'
          }}>
            Review Transactions →
          </button>
        </div>
      )}

      <div className="section-header" style={{ textAlign: 'left', margin: '0 0 32px 0', padding: 0 }}>
        <h2 className="reveal-text" style={{ fontSize: '2.5rem' }}>Overview</h2>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: '24px', marginBottom: '40px' }}>
        
        <div className="bento-card bento-hover">
          <div className="bento-glow"></div>
          <div className="bento-content">
            <div style={{ color: 'var(--text-muted)', fontSize: '0.875rem', fontWeight: 500, marginBottom: '8px' }}>Estimated Balance</div>
            {data ? (
              <>
                <div className="tabular-data" style={{ fontSize: '2.5rem', fontWeight: 500, color: 'var(--text-primary)', display: 'flex', alignItems: 'baseline', gap: '8px', lineHeight: 1.1 }}>
                  {data.estimated_balance.toLocaleString(undefined, { minimumFractionDigits: 2 })} <span style={{ fontSize: '1.25rem', color: 'var(--text-muted)' }}>ETB</span>
                </div>
                {data.week_over_week_pct !== null && data.week_over_week_pct !== undefined ? (
                  <div style={{ marginTop: '16px', display: 'flex', alignItems: 'center', gap: '4px', color: data.week_over_week_pct > 0 ? 'var(--status-danger)' : 'var(--status-success)', fontSize: '0.875rem', fontWeight: 500 }}>
                    {data.week_over_week_pct > 0 ? <ArrowUpRight size={16} /> : <ArrowDownRight size={16} />}
                    {data.week_over_week_pct > 0 ? '+' : ''}{data.week_over_week_pct.toFixed(1)}% spend from last wk
                  </div>
                ) : (
                  <div style={{ marginTop: '16px', display: 'flex', alignItems: 'center', gap: '4px', color: 'var(--text-muted)', fontSize: '0.875rem', fontWeight: 500 }}>
                    No prior week data
                  </div>
                )}
              </>
            ) : (
              <>
                <div className="skeleton-pulse" style={{ height: '40px', width: '200px', marginBottom: '16px' }}></div>
                <div className="skeleton-pulse" style={{ height: '20px', width: '140px' }}></div>
              </>
            )}
          </div>
        </div>

        <div className="bento-card bento-hover">
          <div className="bento-glow"></div>
          <div className="bento-content">
            <div style={{ color: 'var(--text-muted)', fontSize: '0.875rem', fontWeight: 500, marginBottom: '8px' }}>This Week's Income</div>
            {data ? (
              <>
                <div className="tabular-data" style={{ fontSize: '2.5rem', fontWeight: 500, color: 'var(--status-success)', display: 'flex', alignItems: 'baseline', gap: '8px', lineHeight: 1.1 }}>
                  +{data.this_week_income.toLocaleString(undefined, { minimumFractionDigits: 2 })} <span style={{ fontSize: '1.25rem', color: 'var(--text-muted)' }}>ETB</span>
                </div>
                <div style={{ marginTop: '16px', color: 'var(--text-muted)', fontSize: '0.875rem' }}>
                  Total deposits
                </div>
              </>
            ) : (
              <>
                <div className="skeleton-pulse" style={{ height: '40px', width: '180px', marginBottom: '16px' }}></div>
                <div className="skeleton-pulse" style={{ height: '20px', width: '160px' }}></div>
              </>
            )}
          </div>
        </div>

        <div className="bento-card bento-hover">
          <div className="bento-glow"></div>
          <div className="bento-content">
            <div style={{ color: 'var(--text-muted)', fontSize: '0.875rem', fontWeight: 500, marginBottom: '8px' }}>This Week's Spend</div>
            {data ? (
              <>
                <div className="tabular-data" style={{ fontSize: '2.5rem', fontWeight: 500, color: 'var(--status-danger)', display: 'flex', alignItems: 'baseline', gap: '8px', lineHeight: 1.1 }}>
                  −{data.this_week_spend.toLocaleString(undefined, { minimumFractionDigits: 2 })} <span style={{ fontSize: '1.25rem', color: 'var(--text-muted)' }}>ETB</span>
                </div>
                <div style={{ marginTop: '16px', color: 'var(--text-muted)', fontSize: '0.875rem' }}>
                  Top Category: <span style={{ color: 'var(--text-primary)', fontWeight: 500 }}>{data.top_category}</span>
                </div>
              </>
            ) : (
              <>
                <div className="skeleton-pulse" style={{ height: '40px', width: '180px', marginBottom: '16px' }}></div>
                <div className="skeleton-pulse" style={{ height: '20px', width: '160px' }}></div>
              </>
            )}
          </div>
        </div>

      </div>

      <div className="section-header" style={{ textAlign: 'left', margin: '48px 0 24px 0', padding: 0 }}>
        <h2 className="reveal-text" style={{ fontSize: '2rem' }}>Cashflow Overview</h2>
      </div>
      <div className="bento-card" style={{ height: '450px', padding: '24px' }}>
        <div className="bento-content" style={{ height: '100%' }}>
          {data ? (
            chartData.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={chartData} layout="horizontal" margin={{ top: 30, right: 30, left: 30, bottom: 10 }}>
                  <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="rgba(255,255,255,0.05)" />
                  <XAxis dataKey="name" type="category" stroke="var(--text-primary)" fontSize={14} tickLine={false} axisLine={false} />
                  <YAxis type="number" stroke="var(--text-muted)" fontSize={12} tickLine={false} axisLine={false} tickFormatter={(val) => `${val} ETB`} ticks={yTicks} domain={[0, maxTick]} width={80} />
                  <Tooltip 
                    cursor={{ fill: 'rgba(255,255,255,0.02)' }}
                    contentStyle={{ background: 'rgba(10,10,10,0.8)', backdropFilter: 'blur(16px)', border: '1px solid rgba(255,255,255,0.1)', borderRadius: '12px', boxShadow: '0 20px 40px rgba(0,0,0,0.5)' }}
                    itemStyle={{ color: 'var(--text-primary)', fontSize: '0.875rem' }}
                    formatter={(value: any) => [`${Number(value).toLocaleString()} ETB`, 'Total']}
                  />
                  <Bar dataKey="value" radius={[4, 4, 0, 0]} barSize={80}>
                    {chartData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.fill} />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            ) : (
              <div style={{ display: 'flex', height: '100%', alignItems: 'center', justifyContent: 'center', color: 'var(--text-muted)' }}>
                No spending data available.
              </div>
            )
          ) : (
            <div style={{ height: '100%', display: 'flex', flexDirection: 'column', justifyContent: 'center', gap: '24px', padding: '24px 64px' }}>
              <div className="skeleton-pulse" style={{ height: '24px', width: '100%', borderRadius: '4px' }}></div>
              <div className="skeleton-pulse" style={{ height: '24px', width: '85%', borderRadius: '4px' }}></div>
              <div className="skeleton-pulse" style={{ height: '24px', width: '60%', borderRadius: '4px' }}></div>
              <div className="skeleton-pulse" style={{ height: '24px', width: '40%', borderRadius: '4px' }}></div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
