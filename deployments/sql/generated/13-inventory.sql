-- File ini DIHASILKAN oleh scripts/split_db.py. Jangan diedit manual.
CREATE DATABASE IF NOT EXISTS `inventory_db` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE `inventory_db`;
SET FOREIGN_KEY_CHECKS = 0;

-- ===== Tabel milik inventory-service =====
CREATE TABLE IF NOT EXISTS `warehouses` (
    `id` VARCHAR(36) PRIMARY KEY,
    `warehouse_code` VARCHAR(50) NOT NULL UNIQUE,
    `warehouse_name` VARCHAR(150) NOT NULL,
    `address` TEXT NOT NULL,
    `city` VARCHAR(100) NOT NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `warehouse_zones` (
    `id` VARCHAR(36) PRIMARY KEY,
    `warehouse_id` VARCHAR(36) NOT NULL,
    `zone_code` VARCHAR(50) NOT NULL,
    `zone_type` VARCHAR(50) NOT NULL,
    FOREIGN KEY (`warehouse_id`) REFERENCES `warehouses`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `warehouse_shelves` (
    `id` VARCHAR(36) PRIMARY KEY,
    `zone_id` VARCHAR(36) NOT NULL,
    `shelf_code` VARCHAR(50) NOT NULL,
    `capacity_cubic_meter` DECIMAL(8,2) NULL,
    FOREIGN KEY (`zone_id`) REFERENCES `warehouse_zones`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `inventory_items` (
    `id` VARCHAR(36) PRIMARY KEY,
    `product_id` VARCHAR(36) NOT NULL,
    `shelf_id` VARCHAR(36) NULL,
    `serial_number` VARCHAR(100) NULL,
    FOREIGN KEY (`shelf_id`) REFERENCES `warehouse_shelves`(`id`) ON DELETE SET NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `inventory_batches` (
    `id` VARCHAR(36) PRIMARY KEY,
    `batch_number` VARCHAR(100) NOT NULL UNIQUE,
    `production_date` DATE NULL,
    `expiration_date` DATE NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `inventory_stocks` (
    `id` VARCHAR(36) PRIMARY KEY,
    `product_id` VARCHAR(36) NOT NULL,
    `warehouse_id` VARCHAR(36) NOT NULL,
    `batch_id` VARCHAR(36) NULL,
    `quantity_on_hand` INT NOT NULL DEFAULT 0,
    `quantity_reserved` INT NOT NULL DEFAULT 0,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (`warehouse_id`) REFERENCES `warehouses`(`id`) ON DELETE CASCADE,
    FOREIGN KEY (`batch_id`) REFERENCES `inventory_batches`(`id`) ON DELETE SET NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `stock_mutations` (
    `id` VARCHAR(36) PRIMARY KEY,
    `product_id` VARCHAR(36) NOT NULL,
    `source_warehouse_id` VARCHAR(36) NOT NULL,
    `destination_warehouse_id` VARCHAR(36) NOT NULL,
    `quantity` INT NOT NULL,
    `mutation_date` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`source_warehouse_id`) REFERENCES `warehouses`(`id`),
    FOREIGN KEY (`destination_warehouse_id`) REFERENCES `warehouses`(`id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `stock_reservations` (
    `id` VARCHAR(36) PRIMARY KEY,
    `product_id` VARCHAR(36) NOT NULL,
    `reference_order_id` VARCHAR(36) NOT NULL,
    `reserved_quantity` INT NOT NULL,
    `status` ENUM('HOLD', 'CONFIRMED', 'RELEASED') DEFAULT 'HOLD',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `stock_opnames` (
    `id` VARCHAR(36) PRIMARY KEY,
    `warehouse_id` VARCHAR(36) NOT NULL,
    `opname_number` VARCHAR(64) NOT NULL UNIQUE,
    `opname_date` DATE NOT NULL,
    `conducted_by_user_id` VARCHAR(36) NOT NULL,
    FOREIGN KEY (`warehouse_id`) REFERENCES `warehouses`(`id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `stock_opname_items` (
    `id` VARCHAR(36) PRIMARY KEY,
    `stock_opname_id` VARCHAR(36) NOT NULL,
    `product_id` VARCHAR(36) NOT NULL,
    `system_qty` INT NOT NULL,
    `physical_qty` INT NOT NULL,
    `discrepancy_qty` INT NOT NULL,
    FOREIGN KEY (`stock_opname_id`) REFERENCES `stock_opnames`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `supplier_returns` (
    `id` VARCHAR(36) PRIMARY KEY,
    `return_number` VARCHAR(64) NOT NULL UNIQUE,
    `warehouse_id` VARCHAR(36) NOT NULL,
    `return_date` DATE NOT NULL,
    `reason` TEXT NULL,
    FOREIGN KEY (`warehouse_id`) REFERENCES `warehouses`(`id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `supplier_return_items` (
    `id` VARCHAR(36) PRIMARY KEY,
    `supplier_return_id` VARCHAR(36) NOT NULL,
    `product_id` VARCHAR(36) NOT NULL,
    `quantity` INT NOT NULL,
    FOREIGN KEY (`supplier_return_id`) REFERENCES `supplier_returns`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `damaged_goods` (
    `id` VARCHAR(36) PRIMARY KEY,
    `product_id` VARCHAR(36) NOT NULL,
    `warehouse_id` VARCHAR(36) NOT NULL,
    `quantity` INT NOT NULL,
    `damage_description` TEXT NOT NULL,
    `reported_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`warehouse_id`) REFERENCES `warehouses`(`id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `reorder_rules` (
    `id` VARCHAR(36) PRIMARY KEY,
    `product_id` VARCHAR(36) NOT NULL UNIQUE,
    `minimum_threshold` INT NOT NULL,
    `recommended_reorder_qty` INT NOT NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `low_stock_alerts` (
    `id` VARCHAR(36) PRIMARY KEY,
    `product_id` VARCHAR(36) NOT NULL,
    `warehouse_id` VARCHAR(36) NOT NULL,
    `current_stock` INT NOT NULL,
    `threshold` INT NOT NULL,
    `is_resolved` TINYINT(1) DEFAULT 0,
    `alert_time` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`warehouse_id`) REFERENCES `warehouses`(`id`)
) ENGINE=InnoDB;

-- ===== Replika baca-saja (disinkronkan dari service pemilik, JANGAN ditulis oleh service ini) =====
CREATE TABLE IF NOT EXISTS `products` (
    `id` VARCHAR(36) PRIMARY KEY,
    `sku` VARCHAR(64) NOT NULL UNIQUE,
    `title` VARCHAR(255) NOT NULL,
    `description` TEXT NULL,
    `base_price` DECIMAL(15,2) NOT NULL,
    `brand_id` VARCHAR(36) NULL,
    `weight_gram` INT NOT NULL DEFAULT 0,
    `is_published` TINYINT(1) DEFAULT 1,
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP
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
INSERT INTO `warehouses` (`id`, `warehouse_code`, `warehouse_name`, `address`, `city`) VALUES
('w0000001-0000-0000-0000-000000000001', 'WH-DPS-01', 'Hub Logistik Denpasar Barat', 'Jl. Mahendradatta No. 99X', 'Denpasar'),
('w0000001-0000-0000-0000-000000000002', 'WH-JKT-01', 'Central Fulfillment Center Cengkareng', 'Kawasan Pergudangan Soewarna Blok C1', 'Jakarta Barat'),
('w0000001-0000-0000-0000-000000000003', 'WH-SBY-01', 'East Java Distribution Hub Rungkut', 'Jl. Rungkut Industri No. 45', 'Surabaya');
INSERT INTO `warehouse_zones` (`id`, `warehouse_id`, `zone_code`, `zone_type`) VALUES
('z0000001-0000-0000-0000-000000000001', 'w0000001-0000-0000-0000-000000000001', 'Z-AMB-DPS', 'Ambient Dry Storage'),
('z0000001-0000-0000-0000-000000000002', 'w0000001-0000-0000-0000-000000000002', 'Z-ELC-JKT', 'Secure Electronics Zone'),
('z0000001-0000-0000-0000-000000000003', 'w0000001-0000-0000-0000-000000000002', 'Z-FMCG-JKT', 'FMCG High Velocity Area'),
('z0000001-0000-0000-0000-000000000004', 'w0000001-0000-0000-0000-000000000003', 'Z-SBY-MAIN', 'General Merchandising');
INSERT INTO `warehouse_shelves` (`id`, `zone_id`, `shelf_code`, `capacity_cubic_meter`) VALUES
('s0000001-0000-0000-0000-000000000001', 'z0000001-0000-0000-0000-000000000001', 'SHELF-DPS-A1', 50.00),
('s0000001-0000-0000-0000-000000000002', 'z0000001-0000-0000-0000-000000000002', 'SHELF-JKT-E1', 30.00),
('s0000001-0000-0000-0000-000000000003', 'z0000001-0000-0000-0000-000000000003', 'SHELF-JKT-F1', 40.00),
('s0000001-0000-0000-0000-000000000004', 'z0000001-0000-0000-0000-000000000004', 'SHELF-SBY-01', 60.00);
INSERT INTO `inventory_batches` (`id`, `batch_number`, `production_date`, `expiration_date`) VALUES
('bat00001-0000-0000-0000-000000000001', 'BATCH-202608-001', '2026-08-01', '2027-08-01'),
('bat00001-0000-0000-0000-000000000002', 'BATCH-202608-002', '2026-08-10', '2027-02-10');
INSERT INTO `inventory_stocks` (`id`, `product_id`, `warehouse_id`, `batch_id`, `quantity_on_hand`, `quantity_reserved`) VALUES
('stk00001-0000-0000-0000-000000000001', 'p0000001-0000-0000-0000-000000000001', 'w0000001-0000-0000-0000-000000000001', 'bat00001-0000-0000-0000-000000000001', 120, 2),
('stk00001-0000-0000-0000-000000000002', 'p0000001-0000-0000-0000-000000000002', 'w0000001-0000-0000-0000-000000000001', 'bat00001-0000-0000-0000-000000000002', 75, 1),
('stk00001-0000-0000-0000-000000000003', 'p0000001-0000-0000-0000-000000000003', 'w0000001-0000-0000-0000-000000000002', NULL, 200, 0),
('stk00001-0000-0000-0000-000000000004', 'p0000001-0000-0000-0000-000000000004', 'w0000001-0000-0000-0000-000000000002', NULL, 45, 1),
('stk00001-0000-0000-0000-000000000005', 'p0000001-0000-0000-0000-000000000005', 'w0000001-0000-0000-0000-000000000002', NULL, 350, 0),
('stk00001-0000-0000-0000-000000000006', 'p0000001-0000-0000-0000-000000000006', 'w0000001-0000-0000-0000-000000000002', NULL, 400, 0),
('stk00001-0000-0000-0000-000000000007', 'p0000001-0000-0000-0000-000000000008', 'w0000001-0000-0000-0000-000000000002', NULL, 18, 0),
('stk00001-0000-0000-0000-000000000008', 'p0000001-0000-0000-0000-000000000010', 'w0000001-0000-0000-0000-000000000003', NULL, 90, 0);
INSERT INTO `stock_mutations` (`id`, `product_id`, `source_warehouse_id`, `destination_warehouse_id`, `quantity`, `mutation_date`) VALUES
('mut00001-0000-0000-0000-000000000001', 'p0000001-0000-0000-0000-000000000001', 'w0000001-0000-0000-0000-000000000001', 'w0000001-0000-0000-0000-000000000003', 25, '2026-08-28 10:15:00');
INSERT INTO `reorder_rules` (`id`, `product_id`, `minimum_threshold`, `recommended_reorder_qty`) VALUES
('rr000001-0000-0000-0000-000000000001', 'p0000001-0000-0000-0000-000000000001', 30, 100),
('rr000001-0000-0000-0000-000000000002', 'p0000001-0000-0000-0000-000000000004', 20, 50),
('rr000001-0000-0000-0000-000000000003', 'p0000001-0000-0000-0000-000000000008', 5, 20);
INSERT INTO `low_stock_alerts` (`id`, `product_id`, `warehouse_id`, `current_stock`, `threshold`, `is_resolved`) VALUES
('lsa00001-0000-0000-0000-000000000001', 'p0000001-0000-0000-0000-000000000008', 'w0000001-0000-0000-0000-000000000002', 4, 5, 0);
INSERT INTO `stock_opnames` (`id`, `warehouse_id`, `opname_number`, `opname_date`, `conducted_by_user_id`) VALUES
('opn00001-0000-0000-0000-000000000001', 'w0000001-0000-0000-0000-000000000002', 'OPN-202608-JKT', '2026-08-31', 'u0000001-0000-0000-0000-000000000002');
INSERT INTO `stock_opname_items` (`id`, `stock_opname_id`, `product_id`, `system_qty`, `physical_qty`, `discrepancy_qty`) VALUES
('opi00001-0000-0000-0000-000000000001', 'opn00001-0000-0000-0000-000000000001', 'p0000001-0000-0000-0000-000000000003', 200, 200, 0),
('opi00001-0000-0000-0000-000000000002', 'opn00001-0000-0000-0000-000000000001', 'p0000001-0000-0000-0000-000000000004', 46, 45, -1);

SET FOREIGN_KEY_CHECKS = 1;
