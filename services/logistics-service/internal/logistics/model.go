package logistics

import (
	"time"
)

type CourierPartner struct {
	ID          string `db:"id" json:"id"`
	PartnerCode string `db:"partner_code" json:"partner_code"`
	PartnerName string `db:"partner_name" json:"partner_name"`
}

type CourierService struct {
	ID            string `db:"id" json:"id"`
	PartnerID     string `db:"partner_id" json:"partner_id"`
	ServiceName   string `db:"service_name" json:"service_name"`
	EstimatedDays string `db:"estimated_days" json:"estimated_days"`

	// Joined
	PartnerName string `db:"partner_name" json:"partner_name,omitempty"`
}

type ShippingZone struct {
	ID               string `db:"id" json:"id"`
	ZoneName         string `db:"zone_name" json:"zone_name"`
	PostalCodePrefix string `db:"postal_code_prefix" json:"postal_code_prefix"`
}

type ShippingRate struct {
	ID               string  `db:"id" json:"id"`
	CourierServiceID string  `db:"courier_service_id" json:"courier_service_id"`
	OriginZoneID     string  `db:"origin_zone_id" json:"origin_zone_id"`
	DestinationZoneID string `db:"destination_zone_id" json:"destination_zone_id"`
	RatePerKG        float64 `db:"rate_per_kg" json:"rate_per_kg"`

	// Joined
	ServiceName     string `db:"service_name" json:"service_name,omitempty"`
	OriginZoneName  string `db:"origin_zone_name" json:"origin_zone_name,omitempty"`
	DestZoneName    string `db:"dest_zone_name" json:"dest_zone_name,omitempty"`
}

type VehicleFleet struct {
	ID                 string  `db:"id" json:"id"`
	VehiclePlateNumber string  `db:"vehicle_plate_number" json:"vehicle_plate_number"`
	VehicleType        string  `db:"vehicle_type" json:"vehicle_type"`
	CapacityKG         float64 `db:"capacity_kg" json:"capacity_kg"`
}

type CourierDriver struct {
	ID                string  `db:"id" json:"id"`
	DriverName        string  `db:"driver_name" json:"driver_name"`
	LicenseNumber     string  `db:"license_number" json:"license_number"`
	PhoneNumber       string  `db:"phone_number" json:"phone_number"`
	AssignedVehicleID *string `db:"assigned_vehicle_id" json:"assigned_vehicle_id"`

	// Joined
	VehiclePlateNumber *string `db:"vehicle_plate_number" json:"vehicle_plate_number,omitempty"`
}

type ShippingOrder struct {
	ID               string  `db:"id" json:"id"`
	OrderID          string  `db:"order_id" json:"order_id"`
	CourierServiceID string  `db:"courier_service_id" json:"courier_service_id"`
	TrackingNumber   string  `db:"tracking_number" json:"tracking_number"`
	WeightKG         float64 `db:"weight_kg" json:"weight_kg"`
	CurrentStatus    string  `db:"current_status" json:"current_status"`

	// Joined
	OrderNumber  string `db:"order_number" json:"order_number,omitempty"`
	ServiceName  string `db:"service_name" json:"service_name,omitempty"`
	CustomerName string `db:"customer_name" json:"customer_name,omitempty"`
	City         string `db:"city" json:"city,omitempty"`
}

type DeliveryRun struct {
	ID         string    `db:"id" json:"id"`
	RunNumber  string    `db:"run_number" json:"run_number"`
	DriverID   string    `db:"driver_id" json:"driver_id"`
	RunDate    string    `db:"run_date" json:"run_date"`
	Status     string    `db:"status" json:"status"`
	DriverName string    `db:"driver_name" json:"driver_name,omitempty"`
}

type ProofOfDelivery struct {
	ID                string    `db:"id" json:"id"`
	ShippingOrderID   string    `db:"shipping_order_id" json:"shipping_order_id"`
	RecipientName     string    `db:"recipient_name" json:"recipient_name"`
	PhotoProofURL     string    `db:"photo_proof_url" json:"photo_proof_url"`
	SignatureImageURL *string   `db:"signature_image_url" json:"signature_image_url"`
	DeliveredTime     time.Time `db:"delivered_time" json:"delivered_time"`
}

type ShippingTrackingLog struct {
	ID                string    `db:"id" json:"id"`
	ShippingOrderID   string    `db:"shipping_order_id" json:"shipping_order_id"`
	StatusDescription string    `db:"status_description" json:"status_description"`
	Location          string    `db:"location" json:"location"`
	LogTime           time.Time `db:"log_time" json:"log_time"`
}
