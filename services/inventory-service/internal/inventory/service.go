package inventory

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Service interface {
	GetStocks(warehouseID string) ([]InventoryStock, error)
	AdjustStock(stockID string, req AdjustStockRequest) error
	ReserveStock(productID, warehouseID, orderID string, qty int) error
	ReleaseStock(orderID string) error
	CreateMutation(req StockMutationRequest) error
	GetMutations() ([]StockMutation, error)
	GetWarehouses() ([]Warehouse, error)
	CreateWarehouse(req WarehouseRequest) (*Warehouse, error)
	GetWarehouseZones(warehouseID string) ([]WarehouseZone, error)
	GetLowStockAlerts() ([]LowStockAlert, error)
	GetOpnames(warehouseID string) ([]StockOpname, error)
	CreateOpname(userID string, req CreateOpnameRequest) (*StockOpname, error)
}

type inventoryService struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &inventoryService{repo: repo}
}

func (s *inventoryService) GetStocks(warehouseID string) ([]InventoryStock, error) {
	return s.repo.GetStocks(warehouseID)
}

func (s *inventoryService) AdjustStock(stockID string, req AdjustStockRequest) error {
	if req.QuantityOnHand < 0 {
		return errors.New("quantity on hand cannot be negative")
	}
	return s.repo.UpdateStockQuantity(stockID, req.QuantityOnHand)
}

func (s *inventoryService) ReserveStock(productID, warehouseID, orderID string, qty int) error {
	if qty <= 0 {
		return errors.New("quantity must be positive")
	}
	return s.repo.ReserveStock(productID, warehouseID, orderID, qty)
}

func (s *inventoryService) ReleaseStock(orderID string) error {
	return s.repo.ReleaseStock(orderID)
}

func (s *inventoryService) CreateMutation(req StockMutationRequest) error {
	if req.Quantity <= 0 {
		return errors.New("quantity must be greater than zero")
	}
	if req.SourceWarehouseID == req.DestinationWarehouseID {
		return errors.New("source and destination warehouse must be different")
	}

	mutation := &StockMutation{
		ProductID:              req.ProductID,
		SourceWarehouseID:      req.SourceWarehouseID,
		DestinationWarehouseID: req.DestinationWarehouseID,
		Quantity:               req.Quantity,
	}

	return s.repo.CreateStockMutation(mutation)
}

func (s *inventoryService) GetMutations() ([]StockMutation, error) {
	return s.repo.GetStockMutations()
}

func (s *inventoryService) GetWarehouses() ([]Warehouse, error) {
	return s.repo.GetWarehouses()
}

func (s *inventoryService) CreateWarehouse(req WarehouseRequest) (*Warehouse, error) {
	if req.WarehouseCode == "" || req.WarehouseName == "" {
		return nil, errors.New("warehouse_code and warehouse_name are required")
	}
	w := &Warehouse{
		ID:            uuid.NewString(),
		WarehouseCode: req.WarehouseCode,
		WarehouseName: req.WarehouseName,
		Address:       req.Address,
		City:          req.City,
	}
	if err := s.repo.CreateWarehouse(w); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *inventoryService) GetWarehouseZones(warehouseID string) ([]WarehouseZone, error) {
	return s.repo.GetWarehouseZones(warehouseID)
}

func (s *inventoryService) GetLowStockAlerts() ([]LowStockAlert, error) {
	return s.repo.GetLowStockAlerts()
}

func (s *inventoryService) GetOpnames(warehouseID string) ([]StockOpname, error) {
	return s.repo.GetStockOpnames(warehouseID)
}

func (s *inventoryService) CreateOpname(userID string, req CreateOpnameRequest) (*StockOpname, error) {
	opNumber := fmt.Sprintf("OPN-%s-%04d", time.Now().Format("20060102"), time.Now().Unix()%10000)
	opDate := req.OpnameDate
	if opDate == "" {
		opDate = time.Now().Format("2006-01-02")
	}

	op := &StockOpname{
		ID:                uuid.NewString(),
		WarehouseID:       req.WarehouseID,
		OpnameNumber:      opNumber,
		OpnameDate:        opDate,
		ConductedByUserID: userID,
	}

	if err := s.repo.CreateStockOpname(op); err != nil {
		return nil, err
	}
	return op, nil
}
