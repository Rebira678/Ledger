import { useState, useEffect } from 'react';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid, Cell } from 'recharts';
import { ArrowUpRight, AlertCircle } from 'lucide-react';

export function Dashboard() {
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

  const chartData = data ? Object.entries(data.category_totals || {}).map(([name, value]) => ({ name, value })) : [];

  return (
    <div>
      <h1>Overview</h1>
      
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: '24px', marginBottom: '40px' }}>
        
        <div className="glass-card">
          <div style={{ color: 'var(--text-muted)', fontSize: '0.875rem', fontWeight: 500, marginBottom: '8px' }}>Estimated Balance</div>
          {data ? (
            <>
              <div className="tabular-data" style={{ fontSize: '2.25rem', fontWeight: 500, color: 'var(--text-primary)', display: 'flex', alignItems: 'baseline', gap: '8px', lineHeight: 1.1 }}>
                {data.estimated_balance.toLocaleString(undefined, { minimumFractionDigits: 2 })} <span style={{ fontSize: '1rem', color: 'var(--text-muted)' }}>ETB</span>
              </div>
              <div style={{ marginTop: '16px', display: 'flex', alignItems: 'center', gap: '4px', color: 'var(--status-success)', fontSize: '0.875rem', fontWeight: 500 }}>
                <ArrowUpRight size={16} /> +12.5% from last week
              </div>
            </>
          ) : (
            <>
              <div className="skeleton-pulse" style={{ height: '40px', width: '200px', marginBottom: '16px' }}></div>
              <div className="skeleton-pulse" style={{ height: '20px', width: '140px' }}></div>
            </>
          )}
        </div>

        <div className="glass-card">
          <div style={{ color: 'var(--text-muted)', fontSize: '0.875rem', fontWeight: 500, marginBottom: '8px' }}>This Week's Spend</div>
          {data ? (
            <>
              <div className="tabular-data" style={{ fontSize: '2.25rem', fontWeight: 500, color: 'var(--text-primary)', display: 'flex', alignItems: 'baseline', gap: '8px', lineHeight: 1.1 }}>
                {data.this_week_spend.toLocaleString(undefined, { minimumFractionDigits: 2 })} <span style={{ fontSize: '1rem', color: 'var(--text-muted)' }}>ETB</span>
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

        <div className="glass-card" style={{ background: 'var(--status-warning-bg)', borderColor: 'var(--status-warning)' }}>
          <div style={{ color: 'var(--status-warning)', fontSize: '0.875rem', fontWeight: 500, marginBottom: '8px', display: 'flex', alignItems: 'center', gap: '8px' }}>
            <AlertCircle size={16} /> Needs Review
          </div>
          {data ? (
            <>
              <div className="tabular-data" style={{ fontSize: '2.25rem', fontWeight: 500, color: 'var(--status-warning)', marginBottom: '16px', lineHeight: 1.1 }}>
                {data.pending_clarifications}
              </div>
              <button className="secondary" style={{ width: '100%', borderColor: 'var(--status-warning)', color: 'var(--status-warning)' }}>Review Transactions →</button>
            </>
          ) : (
            <>
              <div className="skeleton-pulse" style={{ height: '40px', width: '60px', marginBottom: '16px', background: 'rgba(217, 119, 6, 0.2)' }}></div>
              <div className="skeleton-pulse" style={{ height: '36px', width: '100%', background: 'rgba(217, 119, 6, 0.2)' }}></div>
            </>
          )}
        </div>

      </div>

      <h2>Spending Breakdown</h2>
      <div className="glass-card" style={{ height: '400px', padding: '16px' }}>
        {data ? (
          chartData.length > 0 ? (
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={chartData} layout="vertical" margin={{ top: 10, right: 30, left: 40, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" horizontal={false} stroke="var(--border-subtle)" />
                <XAxis type="number" stroke="var(--text-muted)" fontSize={12} tickLine={false} axisLine={false} tickFormatter={(val) => `${val} ETB`} />
                <YAxis dataKey="name" type="category" stroke="var(--text-primary)" fontSize={12} tickLine={false} axisLine={false} />
                <Tooltip 
                  cursor={{ fill: 'var(--bg-surface-hover)' }}
                  contentStyle={{ background: 'var(--bg-surface)', border: '1px solid var(--border-subtle)', borderRadius: '12px', boxShadow: 'var(--shadow-md)' }}
                  itemStyle={{ color: 'var(--text-primary)', fontSize: '0.875rem' }}
                  formatter={(value: any) => [`${Number(value).toLocaleString()} ETB`, 'Spend']}
                />
                <Bar dataKey="value" radius={[0, 4, 4, 0]} barSize={32}>
                  {chartData.map((_, index) => (
                    <Cell key={`cell-${index}`} fill={['var(--stone-700)', 'var(--pine-600)', 'var(--stone-400)', 'var(--gold-600)'][index % 4]} />
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
            <div className="skeleton-pulse" style={{ height: '32px', width: '100%' }}></div>
            <div className="skeleton-pulse" style={{ height: '32px', width: '85%' }}></div>
            <div className="skeleton-pulse" style={{ height: '32px', width: '60%' }}></div>
            <div className="skeleton-pulse" style={{ height: '32px', width: '40%' }}></div>
          </div>
        )}
      </div>
    </div>
  );
}
