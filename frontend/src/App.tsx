import { useState, useCallback, useEffect } from 'react';
import { ChatWindow } from './components/ChatWindow';
import { InputArea } from './components/InputArea';
import { sendMessage, getSessionHistory } from './api/chat';
import type { Message, SessionState, IntentType } from './types';

function generateId(): string {
  return Math.random().toString(36).substring(2, 15);
}

function HistoryModal({ sessionId, onClose }: { sessionId: string; onClose: () => void }) {
  const [history, setHistory] = useState<Message[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getSessionHistory(sessionId)
      .then((res) => {
        setHistory(res.messages);
      })
      .catch((err) => {
        setError(err.message);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [sessionId]);

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={onClose}>
      <div className="bg-white rounded-lg w-[600px] max-h-[80vh] flex flex-col" onClick={(e) => e.stopPropagation()}>
        <div className="px-4 py-3 border-b flex items-center justify-between">
          <h2 className="font-semibold">会话历史</h2>
          <button onClick={onClose} className="text-gray-500 hover:text-gray-700">✕</button>
        </div>
        <div className="flex-1 overflow-y-auto p-4">
          {loading ? (
            <p className="text-gray-400 text-center">加载中...</p>
          ) : error ? (
            <p className="text-red-500 text-center">{error}</p>
          ) : history.length === 0 ? (
            <p className="text-gray-400 text-center">暂无历史记录</p>
          ) : (
            <div className="space-y-2">
              {history.map((msg, idx) => (
                <div key={idx} className={`p-2 rounded ${msg.role === 'user' ? 'bg-blue-50' : 'bg-gray-50'}`}>
                  <p className="text-xs text-gray-500 mb-1">{msg.role === 'user' ? '你' : '客服'}</p>
                  <p className="text-sm whitespace-pre-wrap">{msg.content}</p>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default function App() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [sessionId] = useState(() => generateId());
  const [userId] = useState(() => 'user-' + generateId(8));
  const [sessionState, setSessionState] = useState<SessionState>();
  const [intentType, setIntentType] = useState<IntentType>();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showHistory, setShowHistory] = useState(false);

  const handleSend = useCallback(async (content: string) => {
    const userMsg: Message = { role: 'user', content, timestamp: new Date().toISOString() };
    setMessages((prev) => [...prev, userMsg]);
    setLoading(true);
    setError(null);

    try {
      const history: Message[] = messages.map((m) => ({
        role: m.role,
        content: m.content,
      }));

      const res = await sendMessage({
        session_id: sessionId,
        message: content,
        user_id: userId,
        history,
      });

      const assistantMsg: Message = {
        role: 'assistant',
        content: res.reply,
        timestamp: new Date().toISOString(),
      };
      setMessages((prev) => [...prev, assistantMsg]);
      setSessionState(res.session_state);
      setIntentType(res.type);
    } catch (err) {
      setError(err instanceof Error ? err.message : '请求失败');
    } finally {
      setLoading(false);
    }
  }, [sessionId, userId, messages]);

  const handleClear = useCallback(() => {
    setMessages([]);
    setSessionState(undefined);
    setIntentType(undefined);
    setError(null);
  }, []);

  return (
    <div className="h-screen flex flex-col bg-gray-100">
      <header className="bg-white border-b px-4 py-3 flex items-center justify-between">
        <h1 className="text-lg font-semibold text-gray-800">AI 智能客服</h1>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowHistory(true)}
            className="text-sm text-gray-500 hover:text-gray-700 px-3 py-1 rounded border hover:bg-gray-50 transition-colors"
          >
            历史记录
          </button>
          <button
            onClick={handleClear}
            className="text-sm text-gray-500 hover:text-gray-700 px-3 py-1 rounded border hover:bg-gray-50 transition-colors"
          >
            新会话
          </button>
        </div>
      </header>

      {error && (
        <div className="bg-red-50 text-red-600 px-4 py-2 text-sm">
          {error}
        </div>
      )}

      <ChatWindow
        messages={messages}
        sessionState={sessionState}
        intentType={intentType}
        loading={loading}
      />

      <InputArea onSend={handleSend} disabled={loading} />

      {showHistory && <HistoryModal sessionId={sessionId} onClose={() => setShowHistory(false)} />}
    </div>
  );
}
