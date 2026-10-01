import { useState, useEffect } from 'react';
import { AlertCircle, CheckCircle2 } from 'lucide-react';

export function Clarifications() {
  const [clars, setClars] = useState<any[]>([]);
  const [inputs, setInputs] = useState<Record<string, string>>({});

  useEffect(() => {
    async function fetchClars() {
      try {
        const res = await fetch('/v1/agent/clarifications', {
          headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
        });
        if (res.ok) {
          const json = await res.json();
          const mapped = (json.items || []).map((c: any) => ({
            id: c.clarification_id,
            question: c.question,
            options: c.options || []
          }));
          setClars(mapped);
        }
      } catch (err) {
        console.error(err);
      }
    }
    fetchClars();
  }, []);

  const handleRespond = async (id: string, responseText: string) => {
    try {
      const res = await fetch(`/v1/agent/clarifications/${id}/respond`, {
        method: 'POST',
        headers: { 
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ answer: responseText })
      });
      if (res.ok) {
        setClars(clars.filter(c => c.id !== id));
      }
    } catch (err) {
      console.error(err);
    }
  };

  return (
    <div>
      <h1>Clarifying Questions</h1>
      
      {clars.length > 0 ? (
        <div style={{ display: 'grid', gap: '20px' }}>
          {clars.map(c => (
            <div key={c.id} className="glass-card" style={{ borderLeft: '4px solid var(--status-warning)', display: 'flex', gap: '24px', alignItems: 'flex-start' }}>
              <div style={{ color: 'var(--status-warning)', marginTop: '4px' }}>
                <AlertCircle size={24} />
              </div>
              <div style={{ flex: 1 }}>
                <p style={{ fontSize: '1rem', margin: '0 0 16px', lineHeight: 1.5, fontWeight: 500, color: 'var(--text-primary)' }}>{c.question}</p>
                <div style={{ display: 'flex', gap: '12px', flexWrap: 'wrap', alignItems: 'center' }}>
                  {c.options.map((opt: string) => (
                    <button 
                      key={opt} 
                      className="secondary" 
                      style={{ padding: '6px 12px' }}
                      onClick={() => handleRespond(c.id, opt)}
                    >
                      {opt}
                    </button>
                  ))}
                  <div style={{ width: '1px', height: '24px', background: 'var(--border-subtle)', margin: '0 8px' }}></div>
                  <input 
                    placeholder="Other category…" 
                    style={{ width: '200px' }} 
                    value={inputs[c.id] || ''}
                    onChange={(e) => setInputs({...inputs, [c.id]: e.target.value})}
                  />
                  <button onClick={() => {
                    if (inputs[c.id]) {
                      handleRespond(c.id, inputs[c.id]);
                    }
                  }}>Save</button>
                </div>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="glass-card" style={{ textAlign: 'center', padding: '64px 24px', borderLeft: '4px solid var(--status-success)' }}>
          <CheckCircle2 size={48} style={{ color: 'var(--status-success)', margin: '0 auto 16px' }} />
          <div style={{ fontSize: '1.25rem', fontWeight: 600, marginBottom: '8px', color: 'var(--text-primary)' }}>Nothing to review!</div>
          <p style={{ color: 'var(--text-muted)', margin: 0, fontSize: '0.875rem' }}>The categorization model is highly confident.</p>
        </div>
      )}
    </div>
  );
}
