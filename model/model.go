package model

type IntentType string

const (
	IntentFAQ     IntentType = "faq"
	IntentFlow    IntentType = "flow"
	IntentUnknown IntentType = "unknown"
)

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

type ChatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
	UserID    string `json:"user_id"`
}

type ChatResponse struct {
	Reply   string       `json:"reply"`
	Type    IntentType   `json:"type"`
	Session SessionState `json:"session_state,omitempty"`
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
	Role    string `json:"role"`
	Content string `json:"content"`
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
