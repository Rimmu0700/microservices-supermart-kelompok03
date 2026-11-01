package order

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/nusantara-supermart/backend/internal/catalog"
	"github.com/nusantara-supermart/backend/internal/inventory"
	"github.com/nusantara-supermart/backend/internal/messaging"
)

type Service interface {
	GetCart(customerID string) (*Cart, error)
	AddToCart(customerID string, req AddToCartRequest) error
	UpdateCartItem(itemID string, req UpdateCartItemRequest) error
	RemoveCartItem(itemID string) error
	Checkout(customerID string, req CheckoutRequest) (*Order, error)
	GetOrders(customerID, userRole, statusID string) ([]Order, error)
	GetOrderByID(id string) (*Order, []OrderItem, *OrderShippingDetail, []OrderStatusHistory, []OrderNote, error)
	UpdateStatus(orderID, statusCode, notes string) error
	CancelOrder(orderID, reason, userID string) error
	AddOrderNote(orderID, userID, content string) error
	GetStatuses() ([]OrderStatus, error)
}

type orderService struct {
	repo         Repository
	catalogSvc   catalog.Service
	inventorySvc inventory.Service
}

func NewService(repo Repository, catalogSvc catalog.Service, inventorySvc inventory.Service) Service {
	return &orderService{
		repo:         repo,
		catalogSvc:   catalogSvc,
		inventorySvc: inventorySvc,
	}
}

func (s *orderService) GetCart(customerID string) (*Cart, error) {
	cart, err := s.repo.GetOrCreateCart(customerID)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.GetCartItems(cart.ID)
	if err != nil {
		return nil, err
	}
	cart.Items = items
	return cart, nil
}

func (s *orderService) AddToCart(customerID string, req AddToCartRequest) error {
	if req.Quantity <= 0 {
		return errors.New("quantity must be greater than zero")
	}

	// Coupling check: verify product exists in Catalog domain
	prod, _, _, err := s.catalogSvc.GetProductByID(req.ProductID)
	if err != nil || prod == nil {
		return errors.New("product does not exist")
	}

	cart, err := s.repo.GetOrCreateCart(customerID)
	if err != nil {
		return err
	}

	return s.repo.AddToCart(cart.ID, req.ProductID, req.Quantity)
}

func (s *orderService) UpdateCartItem(itemID string, req UpdateCartItemRequest) error {
	return s.repo.UpdateCartItem(itemID, req.Quantity)
}

func (s *orderService) RemoveCartItem(itemID string) error {
	return s.repo.RemoveCartItem(itemID)
}

// warehouseRegion mencari kota gudang untuk dipakai sebagai segmen wilayah pada routing key.
func (s *orderService) warehouseRegion(warehouseID string) string {
	list, err := s.inventorySvc.GetWarehouses()
	if err == nil {
		for _, w := range list {
			if w.ID == warehouseID {
				if slug := messaging.Slug(w.City); slug != "" {
					return slug
				}
			}
		}
	}
	return "unknown"
}

func (s *orderService) Checkout(customerID string, req CheckoutRequest) (*Order, error) {
	cart, err := s.GetCart(customerID)
	if err != nil || len(cart.Items) == 0 {
		return nil, errors.New("cart is empty")
	}

	if req.ShippingAddressID == "" || req.WarehouseID == "" {
		return nil, errors.New("shipping_address_id and warehouse_id are required")
	}

	awaitingStatus, err := s.repo.GetOrderStatusByCode("AWAITING_PAYMENT")
	if err != nil {
		return nil, fmt.Errorf("order status AWAITING_PAYMENT not found: %w", err)
	}

	// Calculate totals
	var grossTotal float64
	orderItems := make([]OrderItem, len(cart.Items))
	for i, item := range cart.Items {
		grossTotal += item.Subtotal
		orderItems[i] = OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: item.ProductPrice,
			Subtotal:  item.Subtotal,
		}
	}

	discountAmount := 0.00
	if req.VoucherCode != "" {
		discountAmount = 20000.00 // Standard voucher discount
		if discountAmount > grossTotal {
			discountAmount = grossTotal
		}
	}

	taxAmount := (grossTotal - discountAmount) * 0.11 // PPN 11%
	shippingFee := 15000.00                           // Fixed estimate
	totalNet := (grossTotal - discountAmount) + taxAmount + shippingFee

	orderNumber := fmt.Sprintf("ORD-%s-%04d", time.Now().Format("20060102"), time.Now().Unix()%10000)
	orderID := uuid.NewString()

	newOrder := &Order{
		ID:                orderID,
		OrderNumber:       orderNumber,
		CustomerID:        customerID,
		ShippingAddressID: req.ShippingAddressID,
		OrderStatusID:     awaitingStatus.ID,
		WarehouseID:       req.WarehouseID,
		TotalGrossAmount:  grossTotal,
		DiscountAmount:    discountAmount,
		TaxAmount:         taxAmount,
		ShippingFee:       shippingFee,
		TotalNetAmount:    totalNet,
	}

	courierName := req.CourierName
	if courierName == "" {
		courierName = "GoSend Instant Fulfillment"
	}
	shipping := &OrderShippingDetail{
		OrderID:      orderID,
		CourierName:  courierName,
		ShippingCost: shippingFee,
	}

	// ===== PERUBAHAN TUGAS 3 =====
	// SEBELUM: for tiap item -> inventorySvc.ReserveStock(...)  (sinkron, tight temporal coupling)
	// SEKARANG: Order hanya menyimpan pesanan + event OrderPlaced dalam SATU transaksi (outbox).
	//           Reservasi stok dilakukan Inventory secara asinkron lewat RabbitMQ.
	region := s.warehouseRegion(req.WarehouseID)

	eventItems := make([]messaging.OrderItemData, len(orderItems))
	for i, it := range orderItems {
		eventItems[i] = messaging.OrderItemData{ProductID: it.ProductID, Quantity: it.Quantity}
	}
	ev, err := messaging.NewEvent(messaging.EventOrderPlaced, "order-service", messaging.OrderPlacedData{
		OrderID:        orderID,
		OrderNumber:    orderNumber,
		CustomerID:     customerID,
		WarehouseID:    req.WarehouseID,
		Region:         region,
		TotalNetAmount: int64(math.Round(totalNet)),
		Currency:       "IDR",
		Items:          eventItems,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build OrderPlaced event: %w", err)
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		return nil, fmt.Errorf("failed to encode OrderPlaced event: %w", err)
	}
	outbox := &messaging.OutboxMessage{
		ID:            ev.EventID,
		AggregateType: "order",
		AggregateID:   orderID,
		EventType:     ev.EventType,
		RoutingKey:    fmt.Sprintf("order.%s.placed", region), // <domain>.<wilayah>.<status>
		Payload:       payload,
	}

	// Save order + outbox event (atomik)
	if err := s.repo.CreateOrder(newOrder, orderItems, shipping, outbox); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Clear cart
	_ = s.repo.ClearCart(cart.ID)

	return newOrder, nil
}

func (s *orderService) GetOrders(customerID, userRole, statusID string) ([]Order, error) {
	// If customer, only see own orders
	if userRole == "CUSTOMER" {
		return s.repo.GetOrders(customerID, statusID)
	}
	// Staff/Admin sees all
	return s.repo.GetOrders("", statusID)
}

func (s *orderService) GetOrderByID(id string) (*Order, []OrderItem, *OrderShippingDetail, []OrderStatusHistory, []OrderNote, error) {
	order, err := s.repo.GetOrderByID(id)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	items, _ := s.repo.GetOrderItems(id)
	shipping, _ := s.repo.GetOrderShippingDetail(id)
	histories, _ := s.repo.GetOrderStatusHistories(id)
	notes, _ := s.repo.GetOrderNotes(id)

	return order, items, shipping, histories, notes, nil
}

func (s *orderService) UpdateStatus(orderID, statusCode, notes string) error {
	status, err := s.repo.GetOrderStatusByCode(statusCode)
	if err != nil {
		return fmt.Errorf("status code %s not found", statusCode)
	}
	return s.repo.UpdateOrderStatus(orderID, status.ID, notes)
}

func (s *orderService) CancelOrder(orderID, reason, userID string) error {
	cancelledStatus, err := s.repo.GetOrderStatusByCode("CANCELLED")
	if err != nil {
		return err
	}

	// Coupling: release reserved stock in Inventory domain
	_ = s.inventorySvc.ReleaseStock(orderID)

	return s.repo.CancelOrder(orderID, reason, userID, cancelledStatus.ID)
}

func (s *orderService) AddOrderNote(orderID, userID, content string) error {
	note := &OrderNote{
		OrderID:      orderID,
		AuthorUserID: userID,
		NoteContent:  content,
	}
	return s.repo.AddOrderNote(note)
}

func (s *orderService) GetStatuses() ([]OrderStatus, error) {
	return s.repo.GetOrderStatuses()
}
