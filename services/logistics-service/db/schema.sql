-- File ini DIHASILKAN oleh scripts/split_db.py. Jangan diedit manual.
CREATE DATABASE IF NOT EXISTS `logistics_db` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE `logistics_db`;
SET FOREIGN_KEY_CHECKS = 0;

-- ===== Tabel milik logistics-service =====
CREATE TABLE IF NOT EXISTS `courier_partners` (
    `id` VARCHAR(36) PRIMARY KEY,
    `partner_code` VARCHAR(50) NOT NULL UNIQUE,
    `partner_name` VARCHAR(100) NOT NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `courier_services` (
    `id` VARCHAR(36) PRIMARY KEY,
    `partner_id` VARCHAR(36) NOT NULL,
    `service_name` VARCHAR(100) NOT NULL,
    `estimated_days` VARCHAR(20) NOT NULL,
    FOREIGN KEY (`partner_id`) REFERENCES `courier_partners`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `shipping_zones` (
    `id` VARCHAR(36) PRIMARY KEY,
    `zone_name` VARCHAR(100) NOT NULL UNIQUE,
    `postal_code_prefix` VARCHAR(10) NOT NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `shipping_rates` (
    `id` VARCHAR(36) PRIMARY KEY,
    `courier_service_id` VARCHAR(36) NOT NULL,
    `origin_zone_id` VARCHAR(36) NOT NULL,
    `destination_zone_id` VARCHAR(36) NOT NULL,
    `rate_per_kg` DECIMAL(15,2) NOT NULL,
    FOREIGN KEY (`courier_service_id`) REFERENCES `courier_services`(`id`),
    FOREIGN KEY (`origin_zone_id`) REFERENCES `shipping_zones`(`id`),
    FOREIGN KEY (`destination_zone_id`) REFERENCES `shipping_zones`(`id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `vehicle_fleets` (
    `id` VARCHAR(36) PRIMARY KEY,
    `vehicle_plate_number` VARCHAR(30) NOT NULL UNIQUE,
    `vehicle_type` ENUM('MOTORCYCLE', 'VAN', 'TRUCK') NOT NULL,
    `capacity_kg` DECIMAL(8,2) NOT NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `courier_drivers` (
    `id` VARCHAR(36) PRIMARY KEY,
    `driver_name` VARCHAR(150) NOT NULL,
    `license_number` VARCHAR(50) NOT NULL UNIQUE,
    `phone_number` VARCHAR(30) NOT NULL,
    `assigned_vehicle_id` VARCHAR(36) NULL,
    FOREIGN KEY (`assigned_vehicle_id`) REFERENCES `vehicle_fleets`(`id`) ON DELETE SET NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `shipping_orders` (
    `id` VARCHAR(36) PRIMARY KEY,
    `order_id` VARCHAR(36) NOT NULL UNIQUE,
    `courier_service_id` VARCHAR(36) NOT NULL,
    `tracking_number` VARCHAR(100) NOT NULL UNIQUE,
    `weight_kg` DECIMAL(8,2) NOT NULL,
    `current_status` VARCHAR(50) DEFAULT 'MANIFESTED',
    FOREIGN KEY (`courier_service_id`) REFERENCES `courier_services`(`id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `shipping_manifests` (
    `id` VARCHAR(36) PRIMARY KEY,
    `manifest_code` VARCHAR(64) NOT NULL UNIQUE,
    `warehouse_id` VARCHAR(36) NOT NULL,
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `delivery_runs` (
    `id` VARCHAR(36) PRIMARY KEY,
    `run_number` VARCHAR(64) NOT NULL UNIQUE,
    `driver_id` VARCHAR(36) NOT NULL,
    `run_date` DATE NOT NULL,
    `status` ENUM('ASSIGNED', 'IN_TRANSIT', 'COMPLETED') DEFAULT 'ASSIGNED',
    FOREIGN KEY (`driver_id`) REFERENCES `courier_drivers`(`id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `delivery_run_orders` (
    `delivery_run_id` VARCHAR(36) NOT NULL,
    `shipping_order_id` VARCHAR(36) NOT NULL,
    PRIMARY KEY (`delivery_run_id`, `shipping_order_id`),
    FOREIGN KEY (`delivery_run_id`) REFERENCES `delivery_runs`(`id`) ON DELETE CASCADE,
    FOREIGN KEY (`shipping_order_id`) REFERENCES `shipping_orders`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `proof_of_deliveries` (
    `id` VARCHAR(36) PRIMARY KEY,
    `shipping_order_id` VARCHAR(36) NOT NULL UNIQUE,
    `recipient_name` VARCHAR(150) NOT NULL,
    `signature_image_url` VARCHAR(500) NULL,
    `photo_proof_url` VARCHAR(500) NOT NULL,
    `delivered_time` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`shipping_order_id`) REFERENCES `shipping_orders`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `shipping_tracking_logs` (
    `id` VARCHAR(36) PRIMARY KEY,
    `shipping_order_id` VARCHAR(36) NOT NULL,
    `status_description` VARCHAR(255) NOT NULL,
    `location` VARCHAR(100) NOT NULL,
    `log_time` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`shipping_order_id`) REFERENCES `shipping_orders`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `customs_declarations` (
    `id` VARCHAR(36) PRIMARY KEY,
    `shipping_order_id` VARCHAR(36) NOT NULL UNIQUE,
    `declared_value` DECIMAL(15,2) NOT NULL,
    `hs_code` VARCHAR(30) NOT NULL,
    FOREIGN KEY (`shipping_order_id`) REFERENCES `shipping_orders`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

-- ===== Replika baca-saja (disinkronkan dari service pemilik, JANGAN ditulis oleh service ini) =====
CREATE TABLE IF NOT EXISTS `orders` (
    `id` VARCHAR(36) PRIMARY KEY,
    `order_number` VARCHAR(64) NOT NULL UNIQUE,
    `customer_id` VARCHAR(36) NOT NULL,
    `shipping_address_id` VARCHAR(36) NOT NULL,
    `order_status_id` VARCHAR(36) NOT NULL,
    `warehouse_id` VARCHAR(36) NOT NULL,
    `total_gross_amount` DECIMAL(15,2) NOT NULL,
    `discount_amount` DECIMAL(15,2) DEFAULT 0.00,
    `tax_amount` DECIMAL(15,2) DEFAULT 0.00,
    `shipping_fee` DECIMAL(15,2) NOT NULL,
    `total_net_amount` DECIMAL(15,2) NOT NULL,
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `order_shipping_details` (
    `id` VARCHAR(36) PRIMARY KEY,
    `order_id` VARCHAR(36) NOT NULL UNIQUE,
    `courier_name` VARCHAR(100) NOT NULL,
    `tracking_number` VARCHAR(100) NULL,
    `shipping_cost` DECIMAL(15,2) NOT NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `users` (
    `id` VARCHAR(36) PRIMARY KEY,
    `email` VARCHAR(255) NOT NULL UNIQUE,
    `full_name` VARCHAR(150) NOT NULL,
    `phone_number` VARCHAR(30) NULL,
    `is_active` TINYINT(1) DEFAULT 1,
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `user_addresses` (
    `id` VARCHAR(36) PRIMARY KEY,
    `user_id` VARCHAR(36) NOT NULL,
    `address_label` VARCHAR(50) NOT NULL,
    `recipient_name` VARCHAR(150) NOT NULL,
    `phone_number` VARCHAR(30) NOT NULL,
    `street_address` TEXT NOT NULL,
    `city` VARCHAR(100) NOT NULL,
    `province` VARCHAR(100) NOT NULL,
    `postal_code` VARCHAR(20) NOT NULL,
    `latitude` DECIMAL(10, 8) NULL,
    `longitude` DECIMAL(11, 8) NULL,
    `is_primary` TINYINT(1) DEFAULT 0
) ENGINE=InnoDB;

-- ===== Tabel pendukung event (idempotensi + outbox) =====
CREATE TABLE IF NOT EXISTS `processed_events` (
    `consumer_name` VARCHAR(100) NOT NULL,
    `event_id`      VARCHAR(36)  NOT NULL,
    `event_type`    VARCHAR(100) NOT NULL,
    `processed_at`  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`consumer_name`, `event_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `outbox_events` (
    `id`             VARCHAR(36)  PRIMARY KEY,
    `aggregate_type` VARCHAR(50)  NOT NULL,
    `aggregate_id`   VARCHAR(36)  NOT NULL,
    `event_type`     VARCHAR(100) NOT NULL,
    `routing_key`    VARCHAR(255) NOT NULL,
    `payload`        JSON         NOT NULL,
    `created_at`     DATETIME(3)  DEFAULT CURRENT_TIMESTAMP(3),
    `published_at`   DATETIME(3)  NULL,
    INDEX `idx_outbox_unpublished` (`published_at`, `created_at`)
) ENGINE=InnoDB;

-- ===== Data awal (seed) =====
INSERT INTO `courier_partners` (`id`, `partner_code`, `partner_name`) VALUES
('cp000001-0000-0000-0000-000000000001', 'JNE', 'Jalur Nugraha Ekakurir (JNE)'),
('cp000001-0000-0000-0000-000000000002', 'GOSEND', 'GoSend Instant Fulfillment'),
('cp000001-0000-0000-0000-000000000003', 'NUSANTARA_INTERNAL', 'Armada Logistik Nusantara Express');
INSERT INTO `courier_services` (`id`, `partner_id`, `service_name`, `estimated_days`) VALUES
('cs000001-0000-0000-0000-000000000001', 'cp000001-0000-0000-0000-000000000001', 'JNE REG', '2-3 Hari'),
('cs000001-0000-0000-0000-000000000002', 'cp000001-0000-0000-0000-000000000002', 'Instant Courier', '3 Jam'),
('cs000001-0000-0000-0000-000000000003', 'cp000001-0000-0000-0000-000000000003', 'Same Day Delivery', '6 Jam');
INSERT INTO `shipping_zones` (`id`, `zone_name`, `postal_code_prefix`) VALUES
('sz000001-0000-0000-0000-000000000001', 'DENPASAR_URBAN', '802'),
('sz000001-0000-0000-0000-000000000002', 'JAKARTA_METRO', '12'),
('sz000001-0000-0000-0000-000000000003', 'BANDUNG_CITY', '40');
INSERT INTO `shipping_rates` (`id`, `courier_service_id`, `origin_zone_id`, `destination_zone_id`, `rate_per_kg`) VALUES
('sr000001-0000-0000-0000-000000000001', 'cs000001-0000-0000-0000-000000000002', 'sz000001-0000-0000-0000-000000000001', 'sz000001-0000-0000-0000-000000000001', 3000.00),
('sr000001-0000-0000-0000-000000000002', 'cs000001-0000-0000-0000-000000000001', 'sz000001-0000-0000-0000-000000000002', 'sz000001-0000-0000-0000-000000000003', 11000.00);
INSERT INTO `vehicle_fleets` (`id`, `vehicle_plate_number`, `vehicle_type`, `capacity_kg`) VALUES
('veh00001-0000-0000-0000-000000000001', 'B 9182 ABC', 'VAN', 800.00),
('veh00001-0000-0000-0000-000000000002', 'DK 4501 XY', 'MOTORCYCLE', 60.00);
INSERT INTO `courier_drivers` (`id`, `driver_name`, `license_number`, `phone_number`, `assigned_vehicle_id`) VALUES
('drv00001-0000-0000-0000-000000000001', 'Anto Pratama', 'SIM-C-88192831', '081888223344', 'veh00001-0000-0000-0000-000000000002');
INSERT INTO `shipping_orders` (`id`, `order_id`, `courier_service_id`, `tracking_number`, `weight_kg`, `current_status`) VALUES
('so000001-0000-0000-0000-000000000001', 'ord00001-0000-0000-0000-000000000001', 'cs000001-0000-0000-0000-000000000002', 'GOSEND-DPS-883192', 5.25, 'DELIVERED'),
('so000001-0000-0000-0000-000000000002', 'ord00001-0000-0000-0000-000000000003', 'cs000001-0000-0000-0000-000000000001', 'JNE-CGK-990112', 0.45, 'MANIFESTED');
INSERT INTO `proof_of_deliveries` (`id`, `shipping_order_id`, `recipient_name`, `photo_proof_url`, `delivered_time`) VALUES
('pod00001-0000-0000-0000-000000000001', 'so000001-0000-0000-0000-000000000001', 'Wayan Putra (Diterima Langsung)', 'https://cdn.nusantara.com/pod/pod_ord001.jpg', '2026-09-01 11:45:00');
INSERT INTO `shipping_tracking_logs` (`id`, `shipping_order_id`, `status_description`, `location`, `log_time`) VALUES
('stl00001-0000-0000-0000-000000000001', 'so000001-0000-0000-0000-000000000001', 'Pesanan diserahkan ke driver GoSend', 'Hub Denpasar Barat', '2026-09-01 10:10:00'),
('stl00001-0000-0000-0000-000000000002', 'so000001-0000-0000-0000-000000000001', 'Driver dalam perjalanan ke alamat tujuan', 'Jl. Raya Puputan', '2026-09-01 10:45:00'),
('stl00001-0000-0000-0000-000000000003', 'so000001-0000-0000-0000-000000000001', 'Paket berhasil diterima oleh Wayan Putra', 'Tanjung Bungkak Denpasar', '2026-09-01 11:45:00');

SET FOREIGN_KEY_CHECKS = 1;
