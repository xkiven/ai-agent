import { useState } from 'react';
import ReactMarkdown from 'react-markdown';
import type { Message } from '../types';

interface MessageBubbleProps {
  message: Message;
}

function tryParseJSON(str: string): object | null {
  try {
    const obj = JSON.parse(str);
    if (typeof obj === 'object' && obj !== null) {
      return obj;
    }
    return null;
  } catch {
    return null;
  }
}

function JsonDisplay({ data }: { data: object }) {
  const entries = Object.entries(data);
  return (
    <div className="mt-1">
      <table className="text-sm w-full">
        <tbody>
          {entries.map(([key, value]) => (
            <tr key={key} className="border-b border-gray-100 last:border-0">
              <td className="py-1 font-medium text-gray-600 w-28">{key}</td>
              <td className="py-1 text-gray-800">{String(value)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function MessageBubble({ message }: MessageBubbleProps) {
  const isUser = message.role === 'user';
  const [expanded, setExpanded] = useState(true);
  const jsonData = !isUser ? tryParseJSON(message.content) : null;
  const isJson = jsonData !== null;

  return (
    <div className={`flex ${isUser ? 'justify-end' : 'justify-start'} mb-3`}>
      <div
        className={`max-w-[80%] px-4 py-2 rounded-lg ${
          isUser
            ? 'bg-blue-500 text-white'
            : 'bg-white text-gray-800 shadow-sm'
        }`}
      >
        {isJson ? (
          <>
            <div className="flex items-center gap-2 mb-1">
              <button
                onClick={() => setExpanded(!expanded)}
                className="text-xs text-blue-500 hover:underline"
              >
                {expanded ? '▼ 收起详情' : '▶ 展开详情'}
              </button>
              <span className="text-xs text-gray-400">JSON 格式</span>
            </div>
            {expanded && <JsonDisplay data={jsonData} />}
          </>
        ) : (
          <div className="whitespace-pre-wrap break-words prose prose-sm max-w-none">
            <ReactMarkdown>{message.content}</ReactMarkdown>
          </div>
        )}
        {message.timestamp && (
          <p className={`text-xs mt-1 ${isUser ? 'text-blue-100' : 'text-gray-400'}`}>
            {new Date(message.timestamp).toLocaleTimeString('zh-CN')}
          </p>
        )}
      </div>
    </div>
  );
}
