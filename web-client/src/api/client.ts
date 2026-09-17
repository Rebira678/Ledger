export const API_BASE = '/v1';

function getHeaders() {
  const token = localStorage.getItem('token');
  return {
    'Content-Type': 'application/json',
    ...(token ? { 'Authorization': `Bearer ${token}` } : {})
  };
}

export async function fetchSummary() {
  const res = await fetch(`${API_BASE}/dashboard/summary`, { headers: getHeaders() });
  if (!res.ok) throw new Error('Failed to fetch summary');
  return res.json();
}

export async function fetchTransactions() {
  const res = await fetch(`${API_BASE}/transactions`, { headers: getHeaders() });
  if (!res.ok) throw new Error('Failed to fetch transactions');
  return res.json();
}

export async function fetchClarifications() {
  const res = await fetch(`${API_BASE}/agent/clarifications`, { headers: getHeaders() });
  if (!res.ok) throw new Error('Failed to fetch clarifications');
  return res.json();
}

export async function fetchReports() {
  const res = await fetch(`${API_BASE}/reports/weekly`, { headers: getHeaders() });
  if (!res.ok) throw new Error('Failed to fetch reports');
  return res.json();
}

export async function respondClarification(id: string, answer: string) {
  const res = await fetch(`${API_BASE}/agent/clarifications/${id}/respond`, {
    method: 'POST',
    headers: getHeaders(),
    body: JSON.stringify({ answer })
  });
  if (!res.ok) throw new Error('Failed to respond');
  return res.json();
}
