package support

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Service interface {
	GetCategories() ([]TicketCategory, error)
	GetTickets(userID, role string) ([]CustomerTicket, error)
	GetTicketByID(id string) (*CustomerTicket, []TicketMessage, *TicketRating, error)
	CreateTicket(userID string, req CreateTicketRequest) (*CustomerTicket, error)
	AddMessage(userID, ticketID, message string) (*TicketMessage, error)
	UpdateStatus(ticketID, status string) error
	AssignTicket(ticketID, agentID string) error
	RateTicket(ticketID string, req RateTicketRequest) error
	GetFAQs(categoryID string) ([]FAQArticle, error)
	CreateFAQ(req CreateFAQRequest) (*FAQArticle, error)
	GetDisputes() ([]Dispute, error)
	CreateDispute(req CreateDisputeRequest) (*Dispute, error)
}

type supportService struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &supportService{repo: repo}
}

func (s *supportService) GetCategories() ([]TicketCategory, error) {
	return s.repo.GetTicketCategories()
}

func (s *supportService) GetTickets(userID, role string) ([]CustomerTicket, error) {
	if role == "CUSTOMER" {
		return s.repo.GetTickets(userID)
	}
	return s.repo.GetTickets("")
}

func (s *supportService) GetTicketByID(id string) (*CustomerTicket, []TicketMessage, *TicketRating, error) {
	ticket, err := s.repo.GetTicketByID(id)
	if err != nil {
		return nil, nil, nil, err
	}
	msgs, _ := s.repo.GetTicketMessages(id)
	rating, _ := s.repo.GetTicketRating(id)
	return ticket, msgs, rating, nil
}

func (s *supportService) CreateTicket(userID string, req CreateTicketRequest) (*CustomerTicket, error) {
	if req.Subject == "" || req.CategoryID == "" {
		return nil, errors.New("subject and category_id are required")
	}

	priority := req.Priority
	if priority == "" {
		priority = "MEDIUM"
	}

	tCode := fmt.Sprintf("TKT-%s-%04d", time.Now().Format("20060102"), time.Now().Unix()%10000)
	t := &CustomerTicket{
		ID:         uuid.NewString(),
		TicketCode: tCode,
		CustomerID: userID,
		OrderID:    req.OrderID,
		CategoryID: req.CategoryID,
		Subject:    req.Subject,
		Priority:   priority,
		Status:     "OPEN",
	}

	if err := s.repo.CreateTicket(t, req.Message); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *supportService) AddMessage(userID, ticketID, message string) (*TicketMessage, error) {
	if message == "" {
		return nil, errors.New("message cannot be empty")
	}

	msg := &TicketMessage{
		ID:           uuid.NewString(),
		TicketID:     ticketID,
		SenderUserID: userID,
		MessageBody:  message,
		SentAt:       time.Now(),
	}

	if err := s.repo.CreateTicketMessage(msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func (s *supportService) UpdateStatus(ticketID, status string) error {
	return s.repo.UpdateTicketStatus(ticketID, status)
}

func (s *supportService) AssignTicket(ticketID, agentID string) error {
	return s.repo.AssignTicket(ticketID, agentID)
}

func (s *supportService) RateTicket(ticketID string, req RateTicketRequest) error {
	if req.SatisfactionScore < 1 || req.SatisfactionScore > 5 {
		return errors.New("satisfaction score must be between 1 and 5")
	}

	r := &TicketRating{
		TicketID:          ticketID,
		SatisfactionScore: req.SatisfactionScore,
		Feedback:          req.Feedback,
	}
	return s.repo.RateTicket(r)
}

func (s *supportService) GetFAQs(categoryID string) ([]FAQArticle, error) {
	return s.repo.GetFAQArticles(categoryID)
}

func (s *supportService) CreateFAQ(req CreateFAQRequest) (*FAQArticle, error) {
	faq := &FAQArticle{
		ID:          uuid.NewString(),
		CategoryID:  req.CategoryID,
		Question:    req.Question,
		Answer:      req.Answer,
		IsPublished: true,
	}
	if err := s.repo.CreateFAQArticle(faq); err != nil {
		return nil, err
	}
	return faq, nil
}

func (s *supportService) GetDisputes() ([]Dispute, error) {
	return s.repo.GetDisputes()
}

func (s *supportService) CreateDispute(req CreateDisputeRequest) (*Dispute, error) {
	dNum := fmt.Sprintf("DSP-%s-%04d", time.Now().Format("20060102"), time.Now().Unix()%10000)
	d := &Dispute{
		ID:            uuid.NewString(),
		DisputeNumber: dNum,
		OrderID:       req.OrderID,
		ClaimAmount:   req.ClaimAmount,
		Status:        "OPEN",
	}
	if err := s.repo.CreateDispute(d); err != nil {
		return nil, err
	}
	return d, nil
}
