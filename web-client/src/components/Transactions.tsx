import { Search, Filter } from 'lucide-react';
import { useState, useEffect } from 'react';

export function Transactions() {
  const [txns, setTxns] = useState<any[]>([]);
  const [isMobile, setIsMobile] = useState(window.innerWidth < 768);
  const [, setIsLoading] = useState(true);
  const [editingId, setEditingId] = useState<string | null>(null);

  const categories = ['Groceries', 'Dining', 'Utilities', 'Transport', 'Income', 'Transfer', 'Other'];

  useEffect(() => {
    const handleResize = () => setIsMobile(window.innerWidth < 768);
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  useEffect(() => {
    async function fetchTxns() {
      try {
        const res = await fetch('/v1/transactions', {
          headers: {
            'Authorization': `Bearer ${localStorage.getItem('token')}`
          }
        });
        if (res.ok) {
          const data = await res.json();
          const mapped = (data.transactions || []).map((t: any) => ({
            id: t.transaction_id,
            date: t.occurred_at,
            counterparty: t.counterparty,
            amount: t.amount,
            direction: t.direction,
            category: t.category
          }));
          setTxns(mapped);
        }
      } catch (err) {
        console.error("failed to fetch txns", err);
      } finally {
        setIsLoading(false);
      }
    }
    fetchTxns();
  }, []);

  const handleCategoryChange = async (id: string, newCategory: string) => {
    try {
      const res = await fetch(`/v1/transactions/${id}/category`, {
        method: 'PATCH',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ category: newCategory })
      });
      if (res.ok) {
        setTxns(txns.map(t => t.id === id ? { ...t, category: newCategory } : t));
      }
    } catch (err) {
      console.error("Failed to update category", err);
    } finally {
      setEditingId(null);
    }
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '32px', flexWrap: 'wrap', gap: '16px' }}>
        <div className="section-header" style={{ textAlign: 'left', padding: 0, margin: 0 }}>
          <h2 className="reveal-text" style={{ fontSize: '2.5rem' }}>Transactions</h2>
        </div>
        
        <div style={{ display: 'flex', gap: '16px', flex: isMobile ? 1 : 'none' }}>
          <div style={{ position: 'relative', flex: 1 }}>
            <Search size={18} style={{ position: 'absolute', left: '12px', top: '50%', transform: 'translateY(-50%)', color: 'var(--text-muted)' }} />
            <input placeholder="Search counterparty..." style={{ paddingLeft: '40px' }} />
          </div>
          <button className="secondary">
            <Filter size={16} /> Filter
          </button>
        </div>
      </div>

      <div className="bento-card" style={{ padding: 0, overflow: 'hidden' }}>
        <div className="bento-glow"></div>
        <div className="bento-content" style={{ padding: 0 }}>
        {isMobile ? (
          <div style={{ display: 'flex', flexDirection: 'column' }}>
            {txns.map(t => (
              <div key={t.id} style={{ padding: '16px', borderBottom: '1px solid var(--border-subtle)', display: 'flex', flexDirection: 'column', gap: '12px' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                  <div>
                    <div style={{ fontWeight: 500, color: 'var(--text-primary)' }}>{t.counterparty}</div>
                    <div style={{ color: 'var(--text-muted)', fontSize: '0.75rem' }}>{new Date(t.date).toLocaleDateString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}</div>
                  </div>
                  <div className="tabular-data" style={{ fontWeight: 500, color: t.direction === 'credit' ? 'var(--status-success)' : 'var(--text-primary)' }}>
                    {t.direction === 'credit' ? '+' : '−'}{t.amount.toLocaleString(undefined, { minimumFractionDigits: 2 })}
                  </div>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  {editingId === t.id ? (
                    <select 
                      autoFocus
                      defaultValue={t.category}
                      onChange={(e) => handleCategoryChange(t.id, e.target.value)}
                      onBlur={() => setEditingId(null)}
                      style={{ padding: '4px 8px', fontSize: '0.875rem' }}
                    >
                      {categories.map(c => <option key={c} value={c}>{c}</option>)}
                    </select>
                  ) : t.category ? (
                    <span className="badge neutral">{t.category}</span>
                  ) : (
                    <em style={{ color: 'var(--text-muted)', fontSize: '0.75rem' }}>Unassigned</em>
                  )}
                  <button className="secondary" onClick={() => setEditingId(t.id)} style={{ padding: '4px 12px', fontSize: '0.75rem', borderRadius: '4px', height: '24px' }}>Edit</button>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="table-container">
            <table>
              <thead>
                <tr>
                  <th>Date</th>
                  <th>Counterparty</th>
                  <th style={{ textAlign: 'right' }}>Amount (ETB)</th>
                  <th>Category</th>
                  <th style={{ textAlign: 'right' }}>Action</th>
                </tr>
              </thead>
              <tbody>
                {txns.map(t => (
                  <tr key={t.id}>
                    <td style={{ color: 'var(--text-muted)' }}>{new Date(t.date).toLocaleDateString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}</td>
                    <td style={{ fontWeight: 500, color: 'var(--text-primary)' }}>{t.counterparty}</td>
                    <td className="tabular-data" style={{ textAlign: 'right', fontWeight: 500, color: t.direction === 'credit' ? 'var(--status-success)' : 'var(--text-primary)' }}>
                      {t.direction === 'credit' ? '+' : '−'}{t.amount.toLocaleString(undefined, { minimumFractionDigits: 2 })}
                    </td>
                    <td>
                      {editingId === t.id ? (
                        <select 
                          autoFocus
                          defaultValue={t.category}
                          onChange={(e) => handleCategoryChange(t.id, e.target.value)}
                          onBlur={() => setEditingId(null)}
                          style={{ padding: '4px 8px', fontSize: '0.875rem' }}
                        >
                          {categories.map(c => <option key={c} value={c}>{c}</option>)}
                        </select>
                      ) : t.category ? (
                        <span className="badge neutral">{t.category}</span>
                      ) : (
                        <em style={{ color: 'var(--text-muted)', fontSize: '0.875rem' }}>Unassigned</em>
                      )}
                    </td>
                    <td style={{ textAlign: 'right' }}>
                      <button className="secondary" onClick={() => setEditingId(t.id)} style={{ padding: '4px 12px', fontSize: '0.75rem', borderRadius: '4px', height: '24px' }}>Edit</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        </div>
      </div>
    </div>
  );
}
