package support

import (
	"time"
)

type TicketCategory struct {
	ID           string `db:"id" json:"id"`
	CategoryName string `db:"category_name" json:"category_name"`
}

type CustomerTicket struct {
	ID         string    `db:"id" json:"id"`
	TicketCode string    `db:"ticket_code" json:"ticket_code"`
	CustomerID string    `db:"customer_id" json:"customer_id"`
	OrderID    *string   `db:"order_id" json:"order_id"`
	CategoryID string    `db:"category_id" json:"category_id"`
	Subject    string    `db:"subject" json:"subject"`
	Priority   string    `db:"priority" json:"priority"` // LOW, MEDIUM, HIGH, URGENT
	Status     string    `db:"status" json:"status"`     // OPEN, IN_PROGRESS, RESOLVED, CLOSED
	CreatedAt  time.Time `db:"created_at" json:"created_at"`

	// Joined
	CustomerName string  `db:"customer_name" json:"customer_name,omitempty"`
	CategoryName string  `db:"category_name" json:"category_name,omitempty"`
	OrderNumber  *string `db:"order_number" json:"order_number,omitempty"`
	AgentName    *string `db:"agent_name" json:"agent_name,omitempty"`
}

type TicketMessage struct {
	ID           string    `db:"id" json:"id"`
	TicketID     string    `db:"ticket_id" json:"ticket_id"`
	SenderUserID string    `db:"sender_user_id" json:"sender_user_id"`
	MessageBody  string    `db:"message_body" json:"message_body"`
	SentAt       time.Time `db:"sent_at" json:"sent_at"`

	// Joined
	SenderName string `db:"sender_name" json:"sender_name,omitempty"`
	SenderRole string `db:"sender_role" json:"sender_role,omitempty"`
}

type TicketRating struct {
	ID                string  `db:"id" json:"id"`
	TicketID          string  `db:"ticket_id" json:"ticket_id"`
	SatisfactionScore int     `db:"satisfaction_score" json:"satisfaction_score"`
	Feedback          *string `db:"feedback" json:"feedback"`
}

type FAQArticle struct {
	ID          string `db:"id" json:"id"`
	CategoryID  string `db:"category_id" json:"category_id"`
	Question    string `db:"question" json:"question"`
	Answer      string `db:"answer" json:"answer"`
	IsPublished bool   `db:"is_published" json:"is_published"`

	// Joined
	CategoryName string `db:"category_name" json:"category_name,omitempty"`
}

type Dispute struct {
	ID            string  `db:"id" json:"id"`
	DisputeNumber string  `db:"dispute_number" json:"dispute_number"`
	OrderID       string  `db:"order_id" json:"order_id"`
	ClaimAmount   float64 `db:"claim_amount" json:"claim_amount"`
	Status        string  `db:"status" json:"status"` // OPEN, ARBITRATION, RESOLVED

	// Joined
	OrderNumber string `db:"order_number" json:"order_number,omitempty"`
}
