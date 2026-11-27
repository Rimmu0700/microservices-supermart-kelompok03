package logistics

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
	GetCouriers() ([]CourierPartner, error)
	GetServices() ([]CourierService, error)
	GetRates() ([]ShippingRate, error)
	GetShipments(status string) ([]ShippingOrder, error)
	GetShipmentByID(id string) (*ShippingOrder, []ShippingTrackingLog, *ProofOfDelivery, error)
	GetShipmentByOrderID(orderID string) (*ShippingOrder, []ShippingTrackingLog, *ProofOfDelivery, error)
	CreateShipment(req CreateShippingOrderRequest) (*ShippingOrder, error)
	UpdateStatus(id string, req UpdateShippingStatusRequest) error
	SubmitPOD(shippingOrderID string, req SubmitPODRequest) error
	GetDrivers() ([]CourierDriver, error)
	GetVehicles() ([]VehicleFleet, error)
	GetDeliveryRuns(driverID string) ([]DeliveryRun, error)
}

type logisticsService struct {
	repo     Repository
	orderSvc OrderService
}

func NewService(repo Repository, orderSvc OrderService) Service {
	return &logisticsService{repo: repo, orderSvc: orderSvc}
}

func (s *logisticsService) GetCouriers() ([]CourierPartner, error) {
	return s.repo.GetCouriers()
}

func (s *logisticsService) GetServices() ([]CourierService, error) {
	return s.repo.GetCourierServices()
}

func (s *logisticsService) GetRates() ([]ShippingRate, error) {
	return s.repo.GetRates()
}

func (s *logisticsService) GetShipments(status string) ([]ShippingOrder, error) {
	return s.repo.GetShippingOrders(status)
}

func (s *logisticsService) GetShipmentByID(id string) (*ShippingOrder, []ShippingTrackingLog, *ProofOfDelivery, error) {
	so, err := s.repo.GetShippingOrderByID(id)
	if err != nil {
		return nil, nil, nil, err
	}
	logs, _ := s.repo.GetTrackingLogs(id)
	pod, _ := s.repo.GetPOD(id)
	return so, logs, pod, nil
}

func (s *logisticsService) GetShipmentByOrderID(orderID string) (*ShippingOrder, []ShippingTrackingLog, *ProofOfDelivery, error) {
	so, err := s.repo.GetShippingOrderByOrderID(orderID)
	if err != nil {
		return nil, nil, nil, err
	}
	logs, _ := s.repo.GetTrackingLogs(so.ID)
	pod, _ := s.repo.GetPOD(so.ID)
	return so, logs, pod, nil
}

func (s *logisticsService) CreateShipment(req CreateShippingOrderRequest) (*ShippingOrder, error) {
	trackingNum := fmt.Sprintf("TRK-%s-%04d", time.Now().Format("20060102"), time.Now().Unix()%10000)
	weight := req.WeightKG
	if weight <= 0 {
		weight = 1.0
	}

	so := &ShippingOrder{
		ID:               uuid.NewString(),
		OrderID:          req.OrderID,
		CourierServiceID: req.CourierServiceID,
		TrackingNumber:   trackingNum,
		WeightKG:         weight,
		CurrentStatus:    "MANIFESTED",
	}

	if err := s.repo.CreateShippingOrder(so); err != nil {
		return nil, err
	}

	// Add first log
	_ = s.repo.AddTrackingLog(&ShippingTrackingLog{
		ShippingOrderID:   so.ID,
		StatusDescription: "Paket telah dibuat dan didaftarkan ke manifes logistik",
		Location:          "Hub Pusat",
	})

	return so, nil
}

func (s *logisticsService) UpdateStatus(id string, req UpdateShippingStatusRequest) error {
	if err := s.repo.UpdateShippingStatus(id, req.Status); err != nil {
		return err
	}

	logDesc := req.Description
	if logDesc == "" {
		logDesc = "Status diperbarui menjadi " + req.Status
	}
	location := req.Location
	if location == "" {
		location = "Dalam Perjalanan"
	}

	_ = s.repo.AddTrackingLog(&ShippingTrackingLog{
		ShippingOrderID:   id,
		StatusDescription: logDesc,
		Location:          location,
	})

	// Coupling: if picked up / in transit, mark order as SHIPPED
	if req.Status == "IN_TRANSIT" || req.Status == "PICKED_UP" {
		so, err := s.repo.GetShippingOrderByID(id)
		if err == nil && s.orderSvc != nil {
			_ = s.orderSvc.UpdateStatus(so.OrderID, "SHIPPED", "Paket sedang dikirim oleh kurir")
		}
	}

	return nil
}

func (s *logisticsService) SubmitPOD(shippingOrderID string, req SubmitPODRequest) error {
	if req.RecipientName == "" {
		return errors.New("recipient_name is required")
	}

	photo := req.PhotoProofURL
	if photo == "" {
		photo = "https://cdn.nusantara.com/pod/default_proof.jpg"
	}

	pod := &ProofOfDelivery{
		ShippingOrderID:   shippingOrderID,
		RecipientName:     req.RecipientName,
		PhotoProofURL:     photo,
		SignatureImageURL: req.SignatureImageURL,
	}

	if err := s.repo.SubmitPOD(pod); err != nil {
		return err
	}

	_ = s.repo.UpdateShippingStatus(shippingOrderID, "DELIVERED")

	_ = s.repo.AddTrackingLog(&ShippingTrackingLog{
		ShippingOrderID:   shippingOrderID,
		StatusDescription: fmt.Sprintf("Paket berhasil diterima oleh %s", req.RecipientName),
		Location:          "Alamat Penerima",
	})

	// Coupling: when delivered, update order status to COMPLETED
	so, err := s.repo.GetShippingOrderByID(shippingOrderID)
	if err == nil && s.orderSvc != nil {
		_ = s.orderSvc.UpdateStatus(so.OrderID, "COMPLETED", "Pesanan telah selesai diterima oleh "+req.RecipientName)
	}

	return nil
}

func (s *logisticsService) GetDrivers() ([]CourierDriver, error) {
	return s.repo.GetDrivers()
}

func (s *logisticsService) GetVehicles() ([]VehicleFleet, error) {
	return s.repo.GetVehicles()
}

func (s *logisticsService) GetDeliveryRuns(driverID string) ([]DeliveryRun, error) {
	return s.repo.GetDeliveryRuns(driverID)
}
