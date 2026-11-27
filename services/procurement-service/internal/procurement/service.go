package procurement

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Service interface {
	GetSuppliers() ([]Supplier, error)
	CreateSupplier(req CreateSupplierRequest) (*Supplier, error)
	GetPurchaseOrders() ([]PurchaseOrder, error)
	GetPurchaseOrderByID(id string) (*PurchaseOrder, []PurchaseOrderItem, error)
	CreatePurchaseOrder(req CreatePORequest) (*PurchaseOrder, error)
	ApprovePO(id string) error
	CreateGRN(req CreateGRNRequest) (*GoodsReceiptNote, error)
	GetGRNs(poID string) ([]GoodsReceiptNote, error)
	GetInvoices() ([]PurchaseInvoice, error)
}

type procurementService struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &procurementService{repo: repo}
}

func (s *procurementService) GetSuppliers() ([]Supplier, error) {
	return s.repo.GetSuppliers()
}

func (s *procurementService) CreateSupplier(req CreateSupplierRequest) (*Supplier, error) {
	if req.CompanyName == "" || req.SupplierCode == "" {
		return nil, errors.New("company_name and supplier_code are required")
	}

	sup := &Supplier{
		ID:                      uuid.NewString(),
		SupplierCode:            req.SupplierCode,
		CompanyName:             req.CompanyName,
		TaxIdentificationNumber: req.TaxIdentificationNumber,
		Address:                 req.Address,
	}

	var contact *SupplierContact
	if req.ContactName != "" {
		contact = &SupplierContact{
			ContactName: req.ContactName,
			Email:       req.ContactEmail,
			Phone:       req.ContactPhone,
		}
	}

	if err := s.repo.CreateSupplier(sup, contact); err != nil {
		return nil, err
	}
	return sup, nil
}

func (s *procurementService) GetPurchaseOrders() ([]PurchaseOrder, error) {
	return s.repo.GetPurchaseOrders()
}

func (s *procurementService) GetPurchaseOrderByID(id string) (*PurchaseOrder, []PurchaseOrderItem, error) {
	return s.repo.GetPurchaseOrderByID(id)
}

func (s *procurementService) CreatePurchaseOrder(req CreatePORequest) (*PurchaseOrder, error) {
	if req.SupplierID == "" || req.WarehouseID == "" || len(req.Items) == 0 {
		return nil, errors.New("supplier_id, warehouse_id, and items are required")
	}

	var totalAmount float64
	poItems := make([]PurchaseOrderItem, len(req.Items))
	for i, item := range req.Items {
		subtotal := float64(item.OrderedQty) * item.UnitCost
		totalAmount += subtotal
		poItems[i] = PurchaseOrderItem{
			ProductID:  item.ProductID,
			OrderedQty: item.OrderedQty,
			UnitCost:   item.UnitCost,
		}
	}

	poNum := fmt.Sprintf("PO-%s-%04d", time.Now().Format("20060102"), time.Now().Unix()%10000)
	po := &PurchaseOrder{
		ID:            uuid.NewString(),
		PONumber:      poNum,
		SupplierID:    req.SupplierID,
		WarehouseID:   req.WarehouseID,
		PODate:        time.Now().Format("2006-01-02"),
		TotalPOAmount: totalAmount,
		Status:        "PENDING",
	}

	if err := s.repo.CreatePurchaseOrder(po, poItems); err != nil {
		return nil, err
	}
	return po, nil
}

func (s *procurementService) ApprovePO(id string) error {
	return s.repo.UpdatePOStatus(id, "APPROVED")
}

func (s *procurementService) CreateGRN(req CreateGRNRequest) (*GoodsReceiptNote, error) {
	if req.PurchaseOrderID == "" || len(req.Items) == 0 {
		return nil, errors.New("purchase_order_id and items are required")
	}

	grnNum := fmt.Sprintf("GRN-%s-%04d", time.Now().Format("20060102"), time.Now().Unix()%10000)
	grn := &GoodsReceiptNote{
		ID:              uuid.NewString(),
		GRNNumber:       grnNum,
		PurchaseOrderID: req.PurchaseOrderID,
	}

	grnItems := make([]GoodsReceiptItem, len(req.Items))
	for i, item := range req.Items {
		note := item.Notes
		grnItems[i] = GoodsReceiptItem{
			ProductID:   item.ProductID,
			ReceivedQty: item.ReceivedQty,
			Notes:       &note,
		}
	}

	if err := s.repo.CreateGRN(grn, grnItems); err != nil {
		return nil, err
	}
	return grn, nil
}

func (s *procurementService) GetGRNs(poID string) ([]GoodsReceiptNote, error) {
	return s.repo.GetGRNs(poID)
}

func (s *procurementService) GetInvoices() ([]PurchaseInvoice, error) {
	return s.repo.GetPurchaseInvoices()
}
