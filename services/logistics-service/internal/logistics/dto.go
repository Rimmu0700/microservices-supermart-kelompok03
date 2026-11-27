package logistics

type CreateShippingOrderRequest struct {
	OrderID          string  `json:"order_id"`
	CourierServiceID string  `json:"courier_service_id"`
	WeightKG         float64 `json:"weight_kg"`
}

type UpdateShippingStatusRequest struct {
	Status      string `json:"status"`
	Location    string `json:"location"`
	Description string `json:"description"`
}

type SubmitPODRequest struct {
	RecipientName     string  `json:"recipient_name"`
	PhotoProofURL     string  `json:"photo_proof_url"`
	SignatureImageURL *string `json:"signature_image_url"`
}

type CreateDriverRequest struct {
	DriverName        string  `json:"driver_name"`
	LicenseNumber     string  `json:"license_number"`
	PhoneNumber       string  `json:"phone_number"`
	AssignedVehicleID *string `json:"assigned_vehicle_id"`
}

type CreateRateRequest struct {
	CourierServiceID  string  `json:"courier_service_id"`
	OriginZoneID      string  `json:"origin_zone_id"`
	DestinationZoneID string  `json:"destination_zone_id"`
	RatePerKG         float64 `json:"rate_per_kg"`
}
