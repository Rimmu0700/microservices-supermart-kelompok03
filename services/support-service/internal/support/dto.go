package support

type CreateTicketRequest struct {
	CategoryID string  `json:"category_id"`
	OrderID    *string `json:"order_id"`
	Subject    string  `json:"subject"`
	Priority   string  `json:"priority"` // LOW, MEDIUM, HIGH, URGENT
	Message    string  `json:"message"`
}

type AddMessageRequest struct {
	MessageBody string `json:"message_body"`
}

type AssignTicketRequest struct {
	AgentID string `json:"agent_id"`
}

type UpdateTicketStatusRequest struct {
	Status string `json:"status"` // OPEN, IN_PROGRESS, RESOLVED, CLOSED
}

type RateTicketRequest struct {
	SatisfactionScore int     `json:"satisfaction_score"`
	Feedback          *string `json:"feedback"`
}

type CreateDisputeRequest struct {
	OrderID     string  `json:"order_id"`
	ClaimAmount float64 `json:"claim_amount"`
}

type CreateFAQRequest struct {
	CategoryID string `json:"category_id"`
	Question   string `json:"question"`
	Answer     string `json:"answer"`
}
