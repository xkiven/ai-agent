import { useRef, useEffect } from 'react';
import type { Message, IntentType, SessionState } from '../types';
import { MessageBubble } from './MessageBubble';

interface ChatWindowProps {
  messages: Message[];
  sessionState?: SessionState;
  intentType?: IntentType;
  loading?: boolean;
}

export function ChatWindow({ messages, sessionState, intentType, loading }: ChatWindowProps) {
  const messagesEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const getStateLabel = (state?: SessionState) => {
    const map: Record<SessionState, string> = {
      new: '新会话',
      active: '活跃',
      on_flow: '流程中',
      complete: '已完成',
    };
    return state ? map[state] : '';
  };

  const getIntentLabel = (type?: IntentType) => {
    const map: Record<IntentType, string> = {
      faq: 'FAQ',
      flow: '流程',
      unknown: '未知',
    };
    return type ? map[type] : '';
  };

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-4 px-4 py-2 bg-white border-b text-sm">
        {sessionState && (
          <span className="text-gray-500">
            状态: <span className="font-medium">{getStateLabel(sessionState)}</span>
          </span>
        )}
        {intentType && (
          <span className="text-gray-500">
            类型: <span className="font-medium">{getIntentLabel(intentType)}</span>
          </span>
        )}
      </div>

      <div className="flex-1 overflow-y-auto p-4">
        {messages.length === 0 && (
          <div className="text-center text-gray-400 mt-20">
            <p className="text-lg mb-2">你好！有什么可以帮你的吗？</p>
            <p className="text-sm">发送消息开始对话</p>
          </div>
        )}
        {messages.map((msg, idx) => (
          <MessageBubble key={idx} message={msg} />
        ))}
        {loading && (
          <div className="flex justify-start mb-3">
            <div className="bg-white px-4 py-2 rounded-lg shadow-sm">
              <div className="flex gap-1">
                <span className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '0ms' }}></span>
                <span className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '150ms' }}></span>
                <span className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '300ms' }}></span>
              </div>
            </div>
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>
    </div>
  );
}
