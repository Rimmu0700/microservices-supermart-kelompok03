-- File ini DIHASILKAN oleh scripts/split_db.py. Jangan diedit manual.
CREATE DATABASE IF NOT EXISTS `procurement_db` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE `procurement_db`;
SET FOREIGN_KEY_CHECKS = 0;

-- ===== Tabel milik procurement-service =====
CREATE TABLE IF NOT EXISTS `suppliers` (
    `id` VARCHAR(36) PRIMARY KEY,
    `supplier_code` VARCHAR(50) NOT NULL UNIQUE,
    `company_name` VARCHAR(150) NOT NULL,
    `tax_identification_number` VARCHAR(50) NOT NULL,
    `address` TEXT NOT NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `supplier_contacts` (
    `id` VARCHAR(36) PRIMARY KEY,
    `supplier_id` VARCHAR(36) NOT NULL,
    `contact_name` VARCHAR(150) NOT NULL,
    `email` VARCHAR(255) NOT NULL,
    `phone` VARCHAR(30) NOT NULL,
    FOREIGN KEY (`supplier_id`) REFERENCES `suppliers`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `rfq_requests` (
    `id` VARCHAR(36) PRIMARY KEY,
    `rfq_number` VARCHAR(64) NOT NULL UNIQUE,
    `request_date` DATE NOT NULL,
    `status` ENUM('DRAFT', 'SENT', 'CLOSED') DEFAULT 'DRAFT'
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `rfq_items` (
    `id` VARCHAR(36) PRIMARY KEY,
    `rfq_id` VARCHAR(36) NOT NULL,
    `product_id` VARCHAR(36) NOT NULL,
    `target_quantity` INT NOT NULL,
    FOREIGN KEY (`rfq_id`) REFERENCES `rfq_requests`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `supplier_price_agreements` (
    `id` VARCHAR(36) PRIMARY KEY,
    `supplier_id` VARCHAR(36) NOT NULL,
    `product_id` VARCHAR(36) NOT NULL,
    `contract_unit_price` DECIMAL(15,2) NOT NULL,
    `effective_until` DATE NOT NULL,
    FOREIGN KEY (`supplier_id`) REFERENCES `suppliers`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `purchase_orders` (
    `id` VARCHAR(36) PRIMARY KEY,
    `po_number` VARCHAR(64) NOT NULL UNIQUE,
    `supplier_id` VARCHAR(36) NOT NULL,
    `warehouse_id` VARCHAR(36) NOT NULL,
    `po_date` DATE NOT NULL,
    `total_po_amount` DECIMAL(15,2) NOT NULL,
    `status` ENUM('PENDING', 'APPROVED', 'RECEIVED', 'CANCELLED') DEFAULT 'PENDING',
    FOREIGN KEY (`supplier_id`) REFERENCES `suppliers`(`id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `purchase_order_items` (
    `id` VARCHAR(36) PRIMARY KEY,
    `purchase_order_id` VARCHAR(36) NOT NULL,
    `product_id` VARCHAR(36) NOT NULL,
    `ordered_qty` INT NOT NULL,
    `unit_cost` DECIMAL(15,2) NOT NULL,
    FOREIGN KEY (`purchase_order_id`) REFERENCES `purchase_orders`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `goods_receipt_notes` (
    `id` VARCHAR(36) PRIMARY KEY,
    `grn_number` VARCHAR(64) NOT NULL UNIQUE,
    `purchase_order_id` VARCHAR(36) NOT NULL,
    `received_date` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`purchase_order_id`) REFERENCES `purchase_orders`(`id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `goods_receipt_items` (
    `id` VARCHAR(36) PRIMARY KEY,
    `grn_id` VARCHAR(36) NOT NULL,
    `product_id` VARCHAR(36) NOT NULL,
    `received_qty` INT NOT NULL,
    FOREIGN KEY (`grn_id`) REFERENCES `goods_receipt_notes`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `purchase_invoices` (
    `id` VARCHAR(36) PRIMARY KEY,
    `invoice_number` VARCHAR(64) NOT NULL UNIQUE,
    `purchase_order_id` VARCHAR(36) NOT NULL,
    `invoice_amount` DECIMAL(15,2) NOT NULL,
    `due_date` DATE NOT NULL,
    FOREIGN KEY (`purchase_order_id`) REFERENCES `purchase_orders`(`id`)
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

CREATE TABLE IF NOT EXISTS `warehouses` (
    `id` VARCHAR(36) PRIMARY KEY,
    `warehouse_code` VARCHAR(50) NOT NULL UNIQUE,
    `warehouse_name` VARCHAR(150) NOT NULL,
    `address` TEXT NOT NULL,
    `city` VARCHAR(100) NOT NULL
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
INSERT INTO `suppliers` (`id`, `supplier_code`, `company_name`, `tax_identification_number`, `address`) VALUES
('sup00001-0000-0000-0000-000000000001', 'SUP-IDN-001', 'PT Indofood Sukses Makmur Tbk', '01.001.234.5-092.000', 'Sudirman Plaza, Indofood Tower Lt. 27, Jakarta'),
('sup00001-0000-0000-0000-000000000002', 'SUP-BAL-002', 'Koperasi Tani Kopi Kintamani Bali', '02.441.987.1-901.000', 'Jl. Raya Kintamani No. 7, Bangli, Bali'),
('sup00001-0000-0000-0000-000000000003', 'SUP-LOG-003', 'PT Surya Distribusi Elektronik', '03.882.119.4-015.000', 'Kawasan Industri Pulogadung Blok B, Jakarta Timur');
INSERT INTO `supplier_contacts` (`id`, `supplier_id`, `contact_name`, `email`, `phone`) VALUES
('sc000001-0000-0000-0000-000000000001', 'sup00001-0000-0000-0000-000000000001', 'Hendro Wijaya', 'hendro.w@indofood.co.id', '02157958822'),
('sc000001-0000-0000-0000-000000000002', 'sup00001-0000-0000-0000-000000000002', 'I Gusti Rai', 'gusti.rai@kopikintamani.id', '081238990011');
INSERT INTO `purchase_orders` (`id`, `po_number`, `supplier_id`, `warehouse_id`, `po_date`, `total_po_amount`, `status`) VALUES
('po000001-0000-0000-0000-000000000001', 'PO-2026-0801', 'sup00001-0000-0000-0000-000000000001', 'w0000001-0000-0000-0000-000000000002', '2026-08-15', 7600000.00, 'RECEIVED'),
('po000001-0000-0000-0000-000000000002', 'PO-2026-0901', 'sup00001-0000-0000-0000-000000000002', 'w0000001-0000-0000-0000-000000000001', '2026-09-02', 4500000.00, 'APPROVED');
INSERT INTO `purchase_order_items` (`id`, `purchase_order_id`, `product_id`, `ordered_qty`, `unit_cost`) VALUES
('poi00001-0000-0000-0000-000000000001', 'po000001-0000-0000-0000-000000000001', 'p0000001-0000-0000-0000-000000000001', 100, 76000.00),
('poi00001-0000-0000-0000-000000000002', 'po000001-0000-0000-0000-000000000002', 'p0000001-0000-0000-0000-000000000002', 90, 50000.00);
INSERT INTO `goods_receipt_notes` (`id`, `grn_number`, `purchase_order_id`, `received_date`) VALUES
('grn00001-0000-0000-0000-000000000001', 'GRN-202608-01', 'po000001-0000-0000-0000-000000000001', '2026-08-20 14:00:00');
INSERT INTO `goods_receipt_items` (`id`, `grn_id`, `product_id`, `received_qty`) VALUES
('gri00001-0000-0000-0000-000000000001', 'grn00001-0000-0000-0000-000000000001', 'p0000001-0000-0000-0000-000000000001', 100);
INSERT INTO `purchase_invoices` (`id`, `invoice_number`, `purchase_order_id`, `invoice_amount`, `due_date`) VALUES
('pi000001-0000-0000-0000-000000000001', 'PINV-IND-2026-99', 'po000001-0000-0000-0000-000000000001', 7600000.00, '2026-09-20');

SET FOREIGN_KEY_CHECKS = 1;
