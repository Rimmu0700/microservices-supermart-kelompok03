package payment

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type OrderService interface {
	UpdateStatus(orderID, statusCode, notes string) error
}

type Service interface {
	GetPaymentMethods() ([]PaymentMethod, error)
	GetInvoices(customerID string) ([]PaymentInvoice, error)
	GetInvoiceByID(id string) (*PaymentInvoice, []PaymentTransaction, error)
	GetInvoiceByOrderID(orderID string) (*PaymentInvoice, error)
	CreateInvoice(orderID, customerID string, amount float64) (*PaymentInvoice, error)
	PayInvoice(req PayInvoiceRequest) (*PaymentTransaction, error)
	GetStoreCredit(userID string) (*StoreCredit, []CreditTransaction, error)
	TopupStoreCredit(userID string, amount float64) error
	RequestRefund(userID string, req RefundRequest) (*Refund, error)
	GetRefunds(userID, role string) ([]Refund, error)
	ApproveRefund(refundID string, approved bool) error
}

type paymentService struct {
	repo     Repository
	orderSvc OrderService
}

func NewService(repo Repository, orderSvc OrderService) Service {
	return &paymentService{repo: repo, orderSvc: orderSvc}
}

func (s *paymentService) GetPaymentMethods() ([]PaymentMethod, error) {
	return s.repo.GetPaymentMethods()
}

func (s *paymentService) GetInvoices(customerID string) ([]PaymentInvoice, error) {
	return s.repo.GetInvoices(customerID)
}

func (s *paymentService) GetInvoiceByID(id string) (*PaymentInvoice, []PaymentTransaction, error) {
	inv, err := s.repo.GetInvoiceByID(id)
	if err != nil {
		return nil, nil, err
	}
	txs, _ := s.repo.GetTransactionsByInvoiceID(id)
	return inv, txs, nil
}

func (s *paymentService) GetInvoiceByOrderID(orderID string) (*PaymentInvoice, error) {
	return s.repo.GetInvoiceByOrderID(orderID)
}

func (s *paymentService) CreateInvoice(orderID, customerID string, amount float64) (*PaymentInvoice, error) {
	invNum := fmt.Sprintf("INV-%s-%04d", time.Now().Format("20060102"), time.Now().Unix()%10000)
	inv := &PaymentInvoice{
		ID:            uuid.NewString(),
		InvoiceNumber: invNum,
		OrderID:       orderID,
		CustomerID:    customerID,
		Amount:        amount,
		PaymentStatus: "UNPAID",
		DueDate:       time.Now().Add(24 * time.Hour),
	}
	if err := s.repo.CreateInvoice(inv); err != nil {
		return nil, err
	}
	return inv, nil
}

func (s *paymentService) PayInvoice(req PayInvoiceRequest) (*PaymentTransaction, error) {
	var inv *PaymentInvoice
	var err error

	if req.InvoiceID != "" {
		inv, err = s.repo.GetInvoiceByID(req.InvoiceID)
	}
	if (inv == nil || err != nil) && req.OrderID != "" {
		inv, err = s.repo.GetInvoiceByOrderID(req.OrderID)
	}
	if err != nil || inv == nil {
		return nil, errors.New("invoice not found")
	}

	if inv.PaymentStatus == "PAID" {
		return nil, errors.New("invoice is already paid")
	}

	method, err := s.repo.GetPaymentMethodByID(req.PaymentMethodID)
	if err != nil {
		return nil, errors.New("invalid payment method")
	}

	// Handle store credit payment
	if method.MethodCode == "STORE_CREDIT" {
		desc := fmt.Sprintf("Pembayaran tagihan %s", inv.InvoiceNumber)
		if err := s.repo.AddCreditTransaction(inv.CustomerID, inv.Amount, "DEBIT", desc); err != nil {
			return nil, fmt.Errorf("gagal bayar dengan saldo kredit: %w", err)
		}
	}

	txRef := fmt.Sprintf("TRX-%s-%04d", method.MethodCode, time.Now().Unix()%100000)
	tx := &PaymentTransaction{
		ID:                   uuid.NewString(),
		InvoiceID:            inv.ID,
		PaymentMethodID:      method.ID,
		TransactionReference: txRef,
		AmountPaid:           inv.Amount,
		Status:               "SUCCESS",
	}

	if err := s.repo.RecordTransaction(tx); err != nil {
		return nil, fmt.Errorf("failed to record payment: %w", err)
	}

	// Coupling: notify Order domain that payment was completed
	if s.orderSvc != nil {
		_ = s.orderSvc.UpdateStatus(inv.OrderID, "PROCESSING", "Pembayaran lunas via "+method.MethodName)
	}

	return tx, nil
}

func (s *paymentService) GetStoreCredit(userID string) (*StoreCredit, []CreditTransaction, error) {
	sc, err := s.repo.GetOrCreateStoreCredit(userID)
	if err != nil {
		return nil, nil, err
	}
	txs, _ := s.repo.GetCreditTransactions(userID)
	return sc, txs, nil
}

func (s *paymentService) TopupStoreCredit(userID string, amount float64) error {
	if amount <= 0 {
		return errors.New("topup amount must be greater than zero")
	}
	return s.repo.AddCreditTransaction(userID, amount, "CREDIT", "Topup saldo dompet retail")
}

func (s *paymentService) RequestRefund(userID string, req RefundRequest) (*Refund, error) {
	if req.Amount <= 0 {
		return nil, errors.New("refund amount must be greater than zero")
	}
	refNum := fmt.Sprintf("REF-%s-%04d", time.Now().Format("20060102"), time.Now().Unix()%10000)
	ref := &Refund{
		ID:                uuid.NewString(),
		RefundNumber:      refNum,
		OrderID:           req.OrderID,
		RequestedByUserID: userID,
		TotalRefundAmount: req.Amount,
		RefundStatus:      "PENDING",
	}
	if err := s.repo.CreateRefund(ref); err != nil {
		return nil, err
	}
	return ref, nil
}

func (s *paymentService) GetRefunds(userID, role string) ([]Refund, error) {
	if role == "CUSTOMER" {
		return s.repo.GetRefunds(userID)
	}
	return s.repo.GetRefunds("")
}

func (s *paymentService) ApproveRefund(refundID string, approved bool) error {
	status := "REJECTED"
	if approved {
		status = "APPROVED"
	}
	return s.repo.UpdateRefundStatus(refundID, status)
}
