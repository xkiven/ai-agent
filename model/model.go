package model

type IntentType string

const (
	IntentFAQ     IntentType = "faq"
	IntentFlow    IntentType = "flow"
	IntentUnknown IntentType = "unknown"
)

type IntentDefinition struct {
	ID       string     `yaml:"id"`
	Name     string     `yaml:"name"`
	Type     IntentType `yaml:"type"`
	Enabled  bool       `yaml:"enabled"`
	Priority int        `yaml:"priority"`
	Keywords []string   `yaml:"keywords"`
	Examples []string   `yaml:"examples"`
	NextFlow string     `yaml:"next_flow,omitempty"`
}

type IntentConfig struct {
	Version string             `yaml:"version"`
	Intents []IntentDefinition `yaml:"intents"`
}

type SessionState string

const (
	SessionNew      SessionState = "new"
	SessionActive   SessionState = "active"
	SessionOnFlow   SessionState = "on_flow"
	SessionComplete SessionState = "complete"
)

type TicketStatus string

const (
	TicketOpen     TicketStatus = "open"
	TicketPending  TicketStatus = "pending"
	TicketResolved TicketStatus = "resolved"
	TicketClosed   TicketStatus = "closed"
)

type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleSystem    MessageRole = "system"
)

type DecisionType string

const (
	DecisionContinueFlow DecisionType = "continue_flow"
	DecisionNewIntent    DecisionType = "new_intent"
	DecisionRAG          DecisionType = "rag"
	DecisionTicket       DecisionType = "ticket"
)

type InterruptCheckRequest struct {
	SessionID   string                 `json:"session_id"`
	FlowID      string                 `json:"flow_id"`
	CurrentStep string                 `json:"current_step"`
	UserMessage string                 `json:"user_message"`
	FlowState   map[string]interface{} `json:"flow_state,omitempty"`
}

type InterruptCheckResponse struct {
	ShouldInterrupt bool    `json:"should_interrupt"`
	NewIntent       string  `json:"new_intent,omitempty"`
	Confidence      float64 `json:"confidence"`
	Reason          string  `json:"reason,omitempty"`
}

type DecisionResult struct {
	Type       DecisionType `json:"type"`
	FlowID     string       `json:"flow_id,omitempty"`
	Reply      string       `json:"reply,omitempty"`
	Confidence float64      `json:"confidence"`
}

type ChatRequest struct {
	SessionID string     `json:"session_id"`
	Message   string     `json:"message"`
	UserID    string     `json:"user_id"`
	History   []Message  `json:"history,omitempty"`
	Intent    IntentType `json:"intent,omitempty"`
	FlowID    string     `json:"flow_id,omitempty"`
}

type ChatResponse struct {
	Reply     string       `json:"reply"`
	Type      IntentType   `json:"type"`
	Session   SessionState `json:"session_state,omitempty"`
	SessionID string       `json:"session_id,omitempty"`
	FlowStep  string       `json:"flow_step,omitempty"`
}

type IntentRecognitionRequest struct {
	Message   string    `json:"message"`
	SessionID string    `json:"session_id"`
	History   []Message `json:"history,omitempty"`
}

type IntentRecognitionResponse struct {
	Intent      IntentType `json:"intent"`
	Confidence  float64    `json:"confidence"`
	Reply       string     `json:"reply,omitempty"`
	FlowID      string     `json:"flow_id,omitempty"`
	Suggestions []string   `json:"suggestions,omitempty"`
}

type Message struct {
	Role      MessageRole `json:"role"`
	Content   string      `json:"content"`
	Timestamp string      `json:"timestamp,omitempty"`
}

type Session struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	State       SessionState           `json:"state"`
	Messages    []Message              `json:"messages"`
	FlowID      string                 `json:"flow_id,omitempty"`
	CurrentStep string                 `json:"current_step,omitempty"`
	FlowState   map[string]interface{} `json:"flow_state,omitempty"`
	Version     int64                  `json:"version"` // 版本号，用于乐观锁
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
}

type SessionHistoryResponse struct {
	SessionID string    `json:"session_id"`
	Messages  []Message `json:"messages"`
	Count     int       `json:"count"`
}

type Ticket struct {
	ID          string       `json:"id"`
	SessionID   string       `json:"session_id"`
	UserID      string       `json:"user_id"`
	Intent      IntentType   `json:"intent"`
	Subject     string       `json:"subject"`
	Description string       `json:"description"`
	Status      TicketStatus `json:"status"`
	CreatedAt   string       `json:"created_at"`
	UpdatedAt   string       `json:"updated_at"`
}

type KnowledgeRequest struct {
	Texts    []string         `json:"texts"`
	Metadata []map[string]any `json:"metadata,omitempty"`
}

type KnowledgeResponse struct {
	Success bool   `json:"success"`
	Count   int    `json:"count,omitempty"`
	Message string `json:"message"`
}

type KnowledgeListResponse struct {
	Success bool             `json:"success"`
	Data    []map[string]any `json:"data"`
	Total   int              `json:"total"`
	Message string           `json:"message"`
}
