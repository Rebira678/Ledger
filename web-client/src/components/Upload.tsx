import React, { useState } from 'react';
import { UploadCloud, CheckCircle2, AlertCircle } from 'lucide-react';

export function Upload() {
  const [file, setFile] = useState<File | null>(null);
  const [isUploading, setIsUploading] = useState(false);
  const [status, setStatus] = useState<'idle' | 'success' | 'error'>('idle');
  const [errorMsg, setErrorMsg] = useState('');

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      setFile(e.target.files[0]);
      setStatus('idle');
      setErrorMsg('');
    }
  };

  const handleUpload = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!file) return;

    setIsUploading(true);
    setStatus('idle');
    
    const formData = new FormData();
    formData.append('file', file);
    formData.append('bank_hint', 'AUTO');

    try {
      const response = await fetch('/v1/ingest/statement', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
        },
        body: formData,
      });

      if (!response.ok) {
        const errData = await response.json().catch(() => ({}));
        let serverError = errData.error?.message || errData.message || 'Upload failed';
        
        // Abstract away backend implementation details for the end-user
        if (serverError.includes('exceeds size limit') || serverError.includes('too large')) {
          serverError = 'The uploaded file is too large. Please upload a file smaller than 10MB.';
        } else if (serverError.includes('llm:') || serverError.includes('status 401') || serverError.includes('API key')) {
          serverError = 'Our automated receipt processing service is currently unavailable. Please try again later.';
        } else {
          serverError = "We ran into a little hiccup while processing your file. Please give it another try.";
        }
        
        throw new Error(serverError);
      }

      setStatus('success');
      setFile(null);
    } catch (err: any) {
      setStatus('error');
      setErrorMsg(err.message || "Oops! Something went wrong. Please give it another try.");
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <div className="card" style={{ maxWidth: '600px', margin: '40px auto' }}>
      <h2 style={{ marginBottom: '8px' }}>Upload Receipt / Statement</h2>
      <p style={{ color: 'var(--text-muted)', marginBottom: '32px' }}>
        Upload a receipt image, bank statement PDF, or CSV to automatically extract transaction data.
      </p>

      <form onSubmit={handleUpload}>
        <div style={{ 
          border: '2px dashed var(--border)', 
          borderRadius: '8px', 
          padding: '40px 20px', 
          textAlign: 'center',
          marginBottom: '24px',
          backgroundColor: 'rgba(255, 255, 255, 0.02)'
        }}>
          <UploadCloud size={48} style={{ color: 'var(--primary)', marginBottom: '16px' }} />
          <br />
          <label style={{ cursor: 'pointer', color: 'var(--primary)', fontWeight: '500' }}>
            <span>Browse files</span>
            <input 
              type="file" 
              accept=".pdf,.csv,.png,.jpg,.jpeg" 
              style={{ display: 'none' }} 
              onChange={handleFileChange}
            />
          </label>
          <span style={{ color: 'var(--text-muted)', marginLeft: '8px' }}>or drag and drop</span>
          {file && (
            <div style={{ marginTop: '16px', color: 'var(--text)', fontWeight: '500' }}>
              Selected: {file.name} ({(file.size / 1024).toFixed(1)} KB)
            </div>
          )}
        </div>

        {status === 'success' && (
          <div style={{ display: 'flex', alignItems: 'center', color: 'var(--status-success)', marginBottom: '16px' }}>
            <CheckCircle2 size={18} style={{ marginRight: '8px' }} />
            File successfully uploaded and processed! Check your Transactions.
          </div>
        )}

        {status === 'error' && (
          <div style={{ display: 'flex', alignItems: 'center', color: 'var(--status-danger)', marginBottom: '16px' }}>
            <AlertCircle size={18} style={{ marginRight: '8px' }} />
            {errorMsg}
          </div>
        )}

        <button 
          type="submit" 
          className="btn-primary" 
          disabled={!file || isUploading}
          style={{ width: '100%' }}
        >
          {isUploading ? 'Uploading & Parsing...' : 'Upload File'}
        </button>
      </form>
    </div>
  );
}
