package logistics

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository interface {
	GetCouriers() ([]CourierPartner, error)
	GetCourierServices() ([]CourierService, error)
	GetRates() ([]ShippingRate, error)
	GetShippingOrders(status string) ([]ShippingOrder, error)
	GetShippingOrderByID(id string) (*ShippingOrder, error)
	GetShippingOrderByOrderID(orderID string) (*ShippingOrder, error)
	CreateShippingOrder(so *ShippingOrder) error
	UpdateShippingStatus(id, status string) error
	AddTrackingLog(log *ShippingTrackingLog) error
	GetTrackingLogs(shippingOrderID string) ([]ShippingTrackingLog, error)
	SubmitPOD(pod *ProofOfDelivery) error
	GetPOD(shippingOrderID string) (*ProofOfDelivery, error)
	GetDrivers() ([]CourierDriver, error)
	GetVehicles() ([]VehicleFleet, error)
	GetDeliveryRuns(driverID string) ([]DeliveryRun, error)
}

type mysqlRepository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &mysqlRepository{db: db}
}

func (r *mysqlRepository) GetCouriers() ([]CourierPartner, error) {
	var couriers []CourierPartner
	err := r.db.Select(&couriers, "SELECT * FROM courier_partners ORDER BY partner_name ASC")
	return couriers, err
}

func (r *mysqlRepository) GetCourierServices() ([]CourierService, error) {
	query := `
		SELECT cs.id, cs.partner_id, cs.service_name, cs.estimated_days,
		       cp.partner_name
		FROM courier_services cs
		JOIN courier_partners cp ON cs.partner_id = cp.id
		ORDER BY cp.partner_name ASC, cs.service_name ASC
	`
	var services []CourierService
	err := r.db.Select(&services, query)
	return services, err
}

func (r *mysqlRepository) GetRates() ([]ShippingRate, error) {
	query := `
		SELECT sr.id, sr.courier_service_id, sr.origin_zone_id, sr.destination_zone_id, sr.rate_per_kg,
		       cs.service_name, oz.zone_name AS origin_zone_name, dz.zone_name AS dest_zone_name
		FROM shipping_rates sr
		JOIN courier_services cs ON sr.courier_service_id = cs.id
		JOIN shipping_zones oz ON sr.origin_zone_id = oz.id
		JOIN shipping_zones dz ON sr.destination_zone_id = dz.id
	`
	var rates []ShippingRate
	err := r.db.Select(&rates, query)
	return rates, err
}

func (r *mysqlRepository) GetShippingOrders(status string) ([]ShippingOrder, error) {
	// Monolith seam: Auto-sync any existing orders that do not have a shipping_order yet
	syncQuery := `
		INSERT IGNORE INTO shipping_orders (id, order_id, courier_service_id, tracking_number, weight_kg, current_status)
		SELECT 
			UUID(),
			o.id,
			COALESCE(
				(SELECT cs.id FROM courier_services cs JOIN order_shipping_details osd ON osd.order_id = o.id WHERE osd.courier_name LIKE CONCAT('%', cs.service_name, '%') LIMIT 1),
				'cs000001-0000-0000-0000-000000000002'
			),
			CONCAT('GOSEND-', DATE_FORMAT(o.created_at, '%Y%m%d'), '-', SUBSTRING(o.id, 1, 4)),
			1.50,
			'READY_FOR_PICKUP'
		FROM orders o
		LEFT JOIN shipping_orders so ON o.id = so.order_id
		WHERE so.id IS NULL
	`
	_, _ = r.db.Exec(syncQuery)

	query := `
		SELECT so.id, so.order_id, so.courier_service_id, so.tracking_number, so.weight_kg, so.current_status,
		       o.order_number, cs.service_name, u.full_name AS customer_name,
		       COALESCE(ua.city, '-') AS city
		FROM shipping_orders so
		JOIN orders o ON so.order_id = o.id
		JOIN courier_services cs ON so.courier_service_id = cs.id
		JOIN users u ON o.customer_id = u.id
		LEFT JOIN user_addresses ua ON o.shipping_address_id = ua.id
	`
	args := []interface{}{}
	if status != "" {
		query += " WHERE so.current_status = ?"
		args = append(args, status)
	}
	query += " ORDER BY o.created_at DESC"

	var orders []ShippingOrder
	err := r.db.Select(&orders, query, args...)
	return orders, err
}

func (r *mysqlRepository) GetShippingOrderByID(id string) (*ShippingOrder, error) {
	query := `
		SELECT so.id, so.order_id, so.courier_service_id, so.tracking_number, so.weight_kg, so.current_status,
		       o.order_number, cs.service_name, u.full_name AS customer_name,
		       COALESCE(ua.city, '-') AS city
		FROM shipping_orders so
		JOIN orders o ON so.order_id = o.id
		JOIN courier_services cs ON so.courier_service_id = cs.id
		JOIN users u ON o.customer_id = u.id
		LEFT JOIN user_addresses ua ON o.shipping_address_id = ua.id
		WHERE so.id = ? LIMIT 1
	`
	var so ShippingOrder
	err := r.db.Get(&so, query, id)
	if err != nil {
		return nil, err
	}
	return &so, nil
}

func (r *mysqlRepository) GetShippingOrderByOrderID(orderID string) (*ShippingOrder, error) {
	query := `
		SELECT so.id, so.order_id, so.courier_service_id, so.tracking_number, so.weight_kg, so.current_status,
		       o.order_number, cs.service_name, u.full_name AS customer_name,
		       COALESCE(ua.city, '-') AS city
		FROM shipping_orders so
		JOIN orders o ON so.order_id = o.id
		JOIN courier_services cs ON so.courier_service_id = cs.id
		JOIN users u ON o.customer_id = u.id
		LEFT JOIN user_addresses ua ON o.shipping_address_id = ua.id
		WHERE so.order_id = ? LIMIT 1
	`
	var so ShippingOrder
	err := r.db.Get(&so, query, orderID)
	if err != nil {
		return nil, err
	}
	return &so, nil
}

func (r *mysqlRepository) CreateShippingOrder(so *ShippingOrder) error {
	if so.ID == "" {
		so.ID = uuid.NewString()
	}
	query := `
		INSERT INTO shipping_orders (id, order_id, courier_service_id, tracking_number, weight_kg, current_status)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(query, so.ID, so.OrderID, so.CourierServiceID, so.TrackingNumber, so.WeightKG, so.CurrentStatus)
	return err
}

func (r *mysqlRepository) UpdateShippingStatus(id, status string) error {
	_, err := r.db.Exec("UPDATE shipping_orders SET current_status = ? WHERE id = ?", status, id)
	return err
}

func (r *mysqlRepository) AddTrackingLog(log *ShippingTrackingLog) error {
	if log.ID == "" {
		log.ID = uuid.NewString()
	}
	log.LogTime = time.Now()
	query := `
		INSERT INTO shipping_tracking_logs (id, shipping_order_id, status_description, location, log_time)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(query, log.ID, log.ShippingOrderID, log.StatusDescription, log.Location, log.LogTime)
	return err
}

func (r *mysqlRepository) GetTrackingLogs(shippingOrderID string) ([]ShippingTrackingLog, error) {
	var logs []ShippingTrackingLog
	err := r.db.Select(&logs, "SELECT * FROM shipping_tracking_logs WHERE shipping_order_id = ? ORDER BY log_time ASC", shippingOrderID)
	return logs, err
}

func (r *mysqlRepository) SubmitPOD(pod *ProofOfDelivery) error {
	if pod.ID == "" {
		pod.ID = uuid.NewString()
	}
	pod.DeliveredTime = time.Now()
	query := `
		INSERT INTO proof_of_deliveries (id, shipping_order_id, recipient_name, photo_proof_url, signature_image_url, delivered_time)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE recipient_name = VALUES(recipient_name), photo_proof_url = VALUES(photo_proof_url), delivered_time = VALUES(delivered_time)
	`
	_, err := r.db.Exec(query, pod.ID, pod.ShippingOrderID, pod.RecipientName, pod.PhotoProofURL, pod.SignatureImageURL, pod.DeliveredTime)
	return err
}

func (r *mysqlRepository) GetPOD(shippingOrderID string) (*ProofOfDelivery, error) {
	var pod ProofOfDelivery
	err := r.db.Get(&pod, "SELECT * FROM proof_of_deliveries WHERE shipping_order_id = ? LIMIT 1", shippingOrderID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &pod, err
}

func (r *mysqlRepository) GetDrivers() ([]CourierDriver, error) {
	query := `
		SELECT cd.id, cd.driver_name, cd.license_number, cd.phone_number, cd.assigned_vehicle_id,
		       vf.vehicle_plate_number
		FROM courier_drivers cd
		LEFT JOIN vehicle_fleets vf ON cd.assigned_vehicle_id = vf.id
	`
	var drivers []CourierDriver
	err := r.db.Select(&drivers, query)
	return drivers, err
}

func (r *mysqlRepository) GetVehicles() ([]VehicleFleet, error) {
	var vehicles []VehicleFleet
	err := r.db.Select(&vehicles, "SELECT * FROM vehicle_fleets")
	return vehicles, err
}

func (r *mysqlRepository) GetDeliveryRuns(driverID string) ([]DeliveryRun, error) {
	query := `
		SELECT dr.id, dr.run_number, dr.driver_id, dr.run_date, dr.status,
		       cd.driver_name
		FROM delivery_runs dr
		JOIN courier_drivers cd ON dr.driver_id = cd.id
	`
	args := []interface{}{}
	if driverID != "" {
		query += " WHERE dr.driver_id = ?"
		args = append(args, driverID)
	}
	query += " ORDER BY dr.run_date DESC"

	var runs []DeliveryRun
	err := r.db.Select(&runs, query, args...)
	return runs, err
}
