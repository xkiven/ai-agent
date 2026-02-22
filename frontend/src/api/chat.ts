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

export type StreamCallback = (content: string, type: string) => void;

export async function sendMessageStream(
  req: ChatRequest,
  onChunk: StreamCallback,
  onDone?: () => void,
  onError?: (error: Error) => void
): Promise<void> {
  try {
    const res = await fetch(`${API_BASE}/chat/stream`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(req),
    });

    if (!res.ok) {
      const err = await res.json();
      throw new Error(err.error || '请求失败');
    }

    const reader = res.body?.getReader();
    if (!reader) {
      throw new Error('无法读取响应流');
    }

    const decoder = new TextDecoder();
    let buffer = '';
    const processedContents = new Set<string>();  // 使用 Set 记录已处理的内容

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop() || '';

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const dataStr = line.slice(6);
          // 使用整个 dataStr 作为 key 来去重
          if (processedContents.has(dataStr)) {
            continue;
          }
          processedContents.add(dataStr);
          
          try {
            const data = JSON.parse(dataStr);
            console.log('[FRONTEND DEBUG] received data:', data);
            if (data.type === 'content' && data.content) {
              console.log('[FRONTEND DEBUG] content:', data.content);
              onChunk(data.content, 'content');
            } else if (data.type === 'done') {
              onDone?.();
            } else if (data.error) {
              throw new Error(data.error);
            }
          } catch {
            // JSON 解析失败，跳过该行
            continue;
          }
        }
      }
    }

    onDone?.();
  } catch (err) {
    onError?.(err instanceof Error ? err : new Error('请求失败'));
  }
}
