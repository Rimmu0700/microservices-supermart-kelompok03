package support

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository interface {
	GetTicketCategories() ([]TicketCategory, error)
	GetTickets(customerID string) ([]CustomerTicket, error)
	GetTicketByID(id string) (*CustomerTicket, error)
	CreateTicket(t *CustomerTicket, initialMessage string) error
	UpdateTicketStatus(id, status string) error
	AssignTicket(ticketID, agentID string) error
	GetTicketMessages(ticketID string) ([]TicketMessage, error)
	CreateTicketMessage(msg *TicketMessage) error
	RateTicket(r *TicketRating) error
	GetTicketRating(ticketID string) (*TicketRating, error)
	GetFAQArticles(categoryID string) ([]FAQArticle, error)
	CreateFAQArticle(faq *FAQArticle) error
	GetDisputes() ([]Dispute, error)
	CreateDispute(d *Dispute) error
}

type mysqlRepository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &mysqlRepository{db: db}
}

func (r *mysqlRepository) GetTicketCategories() ([]TicketCategory, error) {
	var cats []TicketCategory
	err := r.db.Select(&cats, "SELECT * FROM ticket_categories ORDER BY category_name ASC")
	return cats, err
}

func (r *mysqlRepository) GetTickets(customerID string) ([]CustomerTicket, error) {
	query := `
		SELECT ct.id, ct.ticket_code, ct.customer_id, ct.order_id, ct.category_id, ct.subject, ct.priority, ct.status, ct.created_at,
		       u.full_name AS customer_name, tc.category_name, o.order_number,
		       agent.full_name AS agent_name
		FROM customer_tickets ct
		JOIN users u ON ct.customer_id = u.id
		JOIN ticket_categories tc ON ct.category_id = tc.id
		LEFT JOIN orders o ON ct.order_id = o.id
		LEFT JOIN ticket_assignments ta ON ct.id = ta.ticket_id
		LEFT JOIN users agent ON ta.assigned_agent_id = agent.id
	`
	args := []interface{}{}
	if customerID != "" {
		query += " WHERE ct.customer_id = ?"
		args = append(args, customerID)
	}
	query += " ORDER BY ct.created_at DESC"

	var tickets []CustomerTicket
	err := r.db.Select(&tickets, query, args...)
	return tickets, err
}

func (r *mysqlRepository) GetTicketByID(id string) (*CustomerTicket, error) {
	query := `
		SELECT ct.id, ct.ticket_code, ct.customer_id, ct.order_id, ct.category_id, ct.subject, ct.priority, ct.status, ct.created_at,
		       u.full_name AS customer_name, tc.category_name, o.order_number,
		       agent.full_name AS agent_name
		FROM customer_tickets ct
		JOIN users u ON ct.customer_id = u.id
		JOIN ticket_categories tc ON ct.category_id = tc.id
		LEFT JOIN orders o ON ct.order_id = o.id
		LEFT JOIN ticket_assignments ta ON ct.id = ta.ticket_id
		LEFT JOIN users agent ON ta.assigned_agent_id = agent.id
		WHERE ct.id = ? LIMIT 1
	`
	var t CustomerTicket
	err := r.db.Get(&t, query, id)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *mysqlRepository) CreateTicket(t *CustomerTicket, initialMessage string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	t.CreatedAt = time.Now()

	insertTicket := `
		INSERT INTO customer_tickets (id, ticket_code, customer_id, order_id, category_id, subject, priority, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = tx.Exec(insertTicket, t.ID, t.TicketCode, t.CustomerID, t.OrderID, t.CategoryID, t.Subject, t.Priority, t.Status, t.CreatedAt)
	if err != nil {
		return err
	}

	if initialMessage != "" {
		msgID := uuid.NewString()
		_, err = tx.Exec(`
			INSERT INTO ticket_messages (id, ticket_id, sender_user_id, message_body, sent_at)
			VALUES (?, ?, ?, ?, ?)
		`, msgID, t.ID, t.CustomerID, initialMessage, t.CreatedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *mysqlRepository) UpdateTicketStatus(id, status string) error {
	_, err := r.db.Exec("UPDATE customer_tickets SET status = ? WHERE id = ?", status, id)
	return err
}

func (r *mysqlRepository) AssignTicket(ticketID, agentID string) error {
	assignID := uuid.NewString()
	query := `
		INSERT INTO ticket_assignments (id, ticket_id, assigned_agent_id, assigned_at)
		VALUES (?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE assigned_agent_id = VALUES(assigned_agent_id), assigned_at = NOW()
	`
	_, err := r.db.Exec(query, assignID, ticketID, agentID)
	return err
}

func (r *mysqlRepository) GetTicketMessages(ticketID string) ([]TicketMessage, error) {
	query := `
		SELECT tm.id, tm.ticket_id, tm.sender_user_id, tm.message_body, tm.sent_at,
		       u.full_name AS sender_name,
		       COALESCE(r.role_name, 'CUSTOMER') AS sender_role
		FROM ticket_messages tm
		JOIN users u ON tm.sender_user_id = u.id
		LEFT JOIN user_roles ur ON u.id = ur.user_id
		LEFT JOIN roles r ON ur.role_id = r.id
		WHERE tm.ticket_id = ?
		ORDER BY tm.sent_at ASC
	`
	var msgs []TicketMessage
	err := r.db.Select(&msgs, query, ticketID)
	return msgs, err
}

func (r *mysqlRepository) CreateTicketMessage(msg *TicketMessage) error {
	if msg.ID == "" {
		msg.ID = uuid.NewString()
	}
	msg.SentAt = time.Now()
	query := `INSERT INTO ticket_messages (id, ticket_id, sender_user_id, message_body, sent_at) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, msg.ID, msg.TicketID, msg.SenderUserID, msg.MessageBody, msg.SentAt)
	return err
}

func (r *mysqlRepository) RateTicket(rat *TicketRating) error {
	if rat.ID == "" {
		rat.ID = uuid.NewString()
	}
	query := `
		INSERT INTO ticket_ratings (id, ticket_id, satisfaction_score, feedback)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE satisfaction_score = VALUES(satisfaction_score), feedback = VALUES(feedback)
	`
	_, err := r.db.Exec(query, rat.ID, rat.TicketID, rat.SatisfactionScore, rat.Feedback)
	return err
}

func (r *mysqlRepository) GetTicketRating(ticketID string) (*TicketRating, error) {
	var rat TicketRating
	err := r.db.Get(&rat, "SELECT * FROM ticket_ratings WHERE ticket_id = ? LIMIT 1", ticketID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &rat, err
}

func (r *mysqlRepository) GetFAQArticles(categoryID string) ([]FAQArticle, error) {
	query := `
		SELECT fa.id, fa.category_id, fa.question, fa.answer, fa.is_published,
		       tc.category_name
		FROM faq_articles fa
		JOIN ticket_categories tc ON fa.category_id = tc.id
		WHERE fa.is_published = 1
	`
	args := []interface{}{}
	if categoryID != "" {
		query += " AND fa.category_id = ?"
		args = append(args, categoryID)
	}
	query += " ORDER BY fa.id ASC"

	var faqs []FAQArticle
	err := r.db.Select(&faqs, query, args...)
	return faqs, err
}

func (r *mysqlRepository) CreateFAQArticle(faq *FAQArticle) error {
	if faq.ID == "" {
		faq.ID = uuid.NewString()
	}
	query := `INSERT INTO faq_articles (id, category_id, question, answer, is_published) VALUES (?, ?, ?, ?, 1)`
	_, err := r.db.Exec(query, faq.ID, faq.CategoryID, faq.Question, faq.Answer)
	return err
}

func (r *mysqlRepository) GetDisputes() ([]Dispute, error) {
	query := `
		SELECT d.id, d.dispute_number, d.order_id, d.claim_amount, d.status,
		       o.order_number
		FROM disputes d
		JOIN orders o ON d.order_id = o.id
		ORDER BY d.id DESC
	`
	var disputes []Dispute
	err := r.db.Select(&disputes, query)
	return disputes, err
}

func (r *mysqlRepository) CreateDispute(d *Dispute) error {
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	query := `INSERT INTO disputes (id, dispute_number, order_id, claim_amount, status) VALUES (?, ?, ?, ?, 'OPEN')`
	_, err := r.db.Exec(query, d.ID, d.DisputeNumber, d.OrderID, d.ClaimAmount)
	return err
}
