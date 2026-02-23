export type MessageRole = 'user' | 'assistant' | 'system';

export type IntentType = 'faq' | 'flow' | 'unknown';

export type SessionState = 'new' | 'active' | 'on_flow' | 'complete';

export interface Message {
  role: MessageRole;
  content: string;
  timestamp?: string;
}

export interface ChatRequest {
  session_id: string;
  message: string;
  user_id: string;
  history?: Message[];
  intent?: IntentType;
  flow_id?: string;
}

export interface ChatResponse {
  reply: string;
  type: IntentType;
  session_state?: SessionState;
  session_id?: string;
  flow_step?: string;
}

export interface IntentRecognitionRequest {
  message: string;
  session_id: string;
  history?: Message[];
}

export interface IntentRecognitionResponse {
  intent: IntentType;
  confidence: number;
  reply?: string;
  flow_id?: string;
  suggestions?: string[];
}

export interface Session {
  id: string;
  user_id: string;
  state: SessionState;
  messages: Message[];
  flow_id?: string;
  current_step?: string;
}

export interface SessionHistoryResponse {
  session_id: string;
  messages: Message[];
  count: number;
}
