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
  if (!res.ok) throw new Error("We're having trouble loading your dashboard summary. Please try refreshing.");
  return res.json();
}

export async function fetchTransactions() {
  const res = await fetch(`${API_BASE}/transactions`, { headers: getHeaders() });
  if (!res.ok) throw new Error("We couldn't load your recent transactions. Please check your connection.");
  return res.json();
}

export async function fetchClarifications() {
  const res = await fetch(`${API_BASE}/agent/clarifications`, { headers: getHeaders() });
  if (!res.ok) throw new Error("We couldn't retrieve your pending questions right now.");
  return res.json();
}

export async function fetchReports() {
  const res = await fetch(`${API_BASE}/reports/weekly`, { headers: getHeaders() });
  if (!res.ok) throw new Error("We're unable to load your weekly reports at the moment.");
  return res.json();
}

export async function respondClarification(id: string, answer: string) {
  const res = await fetch(`${API_BASE}/agent/clarifications/${id}/respond`, {
    method: 'POST',
    headers: getHeaders(),
    body: JSON.stringify({ answer })
  });
  if (!res.ok) throw new Error("We couldn't save your answer. Please try again.");
  return res.json();
}
