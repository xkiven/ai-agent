import type { ChatRequest, ChatResponse, IntentRecognitionRequest, IntentRecognitionResponse, SessionHistoryResponse } from '../types';

const API_BASE = '';

export async function sendMessage(req: ChatRequest): Promise<ChatResponse> {
  const res = await fetch(`${API_BASE}/chat`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error || '请求失败');
  }
  return res.json();
}

export async function recognizeIntent(req: IntentRecognitionRequest): Promise<IntentRecognitionResponse> {
  const res = await fetch(`${API_BASE}/intent/recognize`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error || '请求失败');
  }
  return res.json();
}

export async function getSessionHistory(sessionId: string): Promise<SessionHistoryResponse> {
  const res = await fetch(`${API_BASE}/session/${sessionId}/history`);
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error || '请求失败');
  }
  return res.json();
}

export async function clearSession(sessionId: string): Promise<void> {
  const res = await fetch(`${API_BASE}/session/${sessionId}`, {
    method: 'DELETE',
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error || '请求失败');
  }
}
