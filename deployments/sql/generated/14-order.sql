-- File ini DIHASILKAN oleh scripts/split_db.py. Jangan diedit manual.
CREATE DATABASE IF NOT EXISTS `order_db` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE `order_db`;
SET FOREIGN_KEY_CHECKS = 0;

-- ===== Tabel milik order-service =====
CREATE TABLE IF NOT EXISTS `carts` (
    `id` VARCHAR(36) PRIMARY KEY,
    `user_id` VARCHAR(36) NOT NULL UNIQUE,
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `cart_items` (
    `id` VARCHAR(36) PRIMARY KEY,
    `cart_id` VARCHAR(36) NOT NULL,
    `product_id` VARCHAR(36) NOT NULL,
    `quantity` INT NOT NULL DEFAULT 1,
    FOREIGN KEY (`cart_id`) REFERENCES `carts`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `order_statuses` (
    `id` VARCHAR(36) PRIMARY KEY,
    `status_code` VARCHAR(32) NOT NULL UNIQUE,
    `status_name` VARCHAR(100) NOT NULL
) ENGINE=InnoDB;

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
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`order_status_id`) REFERENCES `order_statuses`(`id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `order_items` (
    `id` VARCHAR(36) PRIMARY KEY,
    `order_id` VARCHAR(36) NOT NULL,
    `product_id` VARCHAR(36) NOT NULL,
    `quantity` INT NOT NULL,
    `unit_price` DECIMAL(15,2) NOT NULL,
    `subtotal` DECIMAL(15,2) NOT NULL,
    FOREIGN KEY (`order_id`) REFERENCES `orders`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `order_status_histories` (
    `id` VARCHAR(36) PRIMARY KEY,
    `order_id` VARCHAR(36) NOT NULL,
    `order_status_id` VARCHAR(36) NOT NULL,
    `notes` TEXT NULL,
    `changed_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`order_id`) REFERENCES `orders`(`id`) ON DELETE CASCADE,
    FOREIGN KEY (`order_status_id`) REFERENCES `order_statuses`(`id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `order_cancellations` (
    `id` VARCHAR(36) PRIMARY KEY,
    `order_id` VARCHAR(36) NOT NULL UNIQUE,
    `cancelled_by_user_id` VARCHAR(36) NOT NULL,
    `cancel_reason` TEXT NOT NULL,
    `cancelled_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`order_id`) REFERENCES `orders`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `order_discounts` (
    `id` VARCHAR(36) PRIMARY KEY,
    `order_id` VARCHAR(36) NOT NULL,
    `discount_type` VARCHAR(50) NOT NULL,
    `discount_value` DECIMAL(15,2) NOT NULL,
    FOREIGN KEY (`order_id`) REFERENCES `orders`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `order_tax_details` (
    `id` VARCHAR(36) PRIMARY KEY,
    `order_id` VARCHAR(36) NOT NULL,
    `tax_name` VARCHAR(50) NOT NULL,
    `tax_rate_percent` DECIMAL(5,2) NOT NULL,
    `tax_calculated_amount` DECIMAL(15,2) NOT NULL,
    FOREIGN KEY (`order_id`) REFERENCES `orders`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `order_shipping_details` (
    `id` VARCHAR(36) PRIMARY KEY,
    `order_id` VARCHAR(36) NOT NULL UNIQUE,
    `courier_name` VARCHAR(100) NOT NULL,
    `tracking_number` VARCHAR(100) NULL,
    `shipping_cost` DECIMAL(15,2) NOT NULL,
    FOREIGN KEY (`order_id`) REFERENCES `orders`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `checkout_sessions` (
    `id` VARCHAR(36) PRIMARY KEY,
    `user_id` VARCHAR(36) NOT NULL,
    `session_payload` JSON NOT NULL,
    `expires_at` DATETIME NOT NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `sales_quotes` (
    `id` VARCHAR(36) PRIMARY KEY,
    `quote_number` VARCHAR(64) NOT NULL UNIQUE,
    `customer_id` VARCHAR(36) NOT NULL,
    `total_amount` DECIMAL(15,2) NOT NULL,
    `valid_until` DATE NOT NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `sales_quote_items` (
    `id` VARCHAR(36) PRIMARY KEY,
    `quote_id` VARCHAR(36) NOT NULL,
    `product_id` VARCHAR(36) NOT NULL,
    `quantity` INT NOT NULL,
    `quoted_price` DECIMAL(15,2) NOT NULL,
    FOREIGN KEY (`quote_id`) REFERENCES `sales_quotes`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `split_orders` (
    `id` VARCHAR(36) PRIMARY KEY,
    `parent_order_id` VARCHAR(36) NOT NULL,
    `child_order_id` VARCHAR(36) NOT NULL,
    `split_reason` VARCHAR(255) NULL,
    FOREIGN KEY (`parent_order_id`) REFERENCES `orders`(`id`) ON DELETE CASCADE,
    FOREIGN KEY (`child_order_id`) REFERENCES `orders`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `preorders` (
    `id` VARCHAR(36) PRIMARY KEY,
    `order_id` VARCHAR(36) NOT NULL UNIQUE,
    `expected_availability_date` DATE NOT NULL,
    FOREIGN KEY (`order_id`) REFERENCES `orders`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `order_notes` (
    `id` VARCHAR(36) PRIMARY KEY,
    `order_id` VARCHAR(36) NOT NULL,
    `author_user_id` VARCHAR(36) NOT NULL,
    `note_content` TEXT NOT NULL,
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`order_id`) REFERENCES `orders`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `order_audit_logs` (
    `id` VARCHAR(36) PRIMARY KEY,
    `order_id` VARCHAR(36) NOT NULL,
    `action_taken` VARCHAR(100) NOT NULL,
    `raw_changes` JSON NULL,
    `timestamp` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`order_id`) REFERENCES `orders`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `order_item_fulfillments` (
    `id` VARCHAR(36) PRIMARY KEY,
    `order_item_id` VARCHAR(36) NOT NULL,
    `fulfilled_quantity` INT NOT NULL,
    `fulfilled_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`order_item_id`) REFERENCES `order_items`(`id`) ON DELETE CASCADE
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

CREATE TABLE IF NOT EXISTS `product_images` (
    `id` VARCHAR(36) PRIMARY KEY,
    `product_id` VARCHAR(36) NOT NULL,
    `image_url` VARCHAR(500) NOT NULL,
    `is_primary` TINYINT(1) DEFAULT 0,
    `display_order` INT DEFAULT 0
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
INSERT INTO `order_statuses` (`id`, `status_code`, `status_name`) VALUES
('os000001-0000-0000-0000-000000000001', 'AWAITING_PAYMENT', 'Menunggu Pembayaran'),
('os000001-0000-0000-0000-000000000002', 'PROCESSING', 'Pembayaran Lunas, Sedang Dikemas'),
('os000001-0000-0000-0000-000000000003', 'SHIPPED', 'Pesanan Telah Diserahkan ke Kurir'),
('os000001-0000-0000-0000-000000000004', 'COMPLETED', 'Pesanan Selesai Diterima'),
('os000001-0000-0000-0000-000000000005', 'CANCELLED', 'Pesanan Dibatalkan');
INSERT INTO `carts` (`id`, `user_id`) VALUES
('cart0001-0000-0000-0000-000000000001', 'u0000001-0000-0000-0000-000000000008');
INSERT INTO `cart_items` (`id`, `cart_id`, `product_id`, `quantity`) VALUES
('ci000001-0000-0000-0000-000000000001', 'cart0001-0000-0000-0000-000000000001', 'p0000001-0000-0000-0000-000000000002', 2),
('ci000001-0000-0000-0000-000000000002', 'cart0001-0000-0000-0000-000000000001', 'p0000001-0000-0000-0000-000000000005', 1);
INSERT INTO `orders` (
    `id`, `order_number`, `customer_id`, `shipping_address_id`, `order_status_id`, `warehouse_id`, 
    `total_gross_amount`, `discount_amount`, `tax_amount`, `shipping_fee`, `total_net_amount`, `created_at`
) VALUES (
    'ord00001-0000-0000-0000-000000000001', 'ORD-2026-0901-001', 
    'u0000001-0000-0000-0000-000000000003', 'a0000001-0000-0000-0000-000000000001', 
    'os000001-0000-0000-0000-000000000004', 'w0000001-0000-0000-0000-000000000001',
    258000.00, 20000.00, 26180.00, 15000.00, 279180.00, '2026-09-01 09:30:00'
);
INSERT INTO `order_items` (`id`, `order_id`, `product_id`, `quantity`, `unit_price`, `subtotal`) VALUES
('oi000001-0000-0000-0000-000000000001', 'ord00001-0000-0000-0000-000000000001', 'p0000001-0000-0000-0000-000000000001', 2, 95000.00, 190000.00),
('oi000001-0000-0000-0000-000000000002', 'ord00001-0000-0000-0000-000000000001', 'p0000001-0000-0000-0000-000000000002', 1, 68000.00, 68000.00);
INSERT INTO `order_status_histories` (`id`, `order_id`, `order_status_id`, `notes`) VALUES
('osh00001-0000-0000-0000-000000000001', 'ord00001-0000-0000-0000-000000000001', 'os000001-0000-0000-0000-000000000001', 'Pesanan dibuat pelanggan'),
('osh00001-0000-0000-0000-000000000002', 'ord00001-0000-0000-0000-000000000001', 'os000001-0000-0000-0000-000000000002', 'Pembayaran QRIS lunas diverifikasi'),
('osh00001-0000-0000-0000-000000000003', 'ord00001-0000-0000-0000-000000000001', 'os000001-0000-0000-0000-000000000003', 'Barang dipickup kurir GoSend'),
('osh00001-0000-0000-0000-000000000004', 'ord00001-0000-0000-0000-000000000001', 'os000001-0000-0000-0000-000000000004', 'Pesanan sukses sampai di alamat');
INSERT INTO `orders` (
    `id`, `order_number`, `customer_id`, `shipping_address_id`, `order_status_id`, `warehouse_id`, 
    `total_gross_amount`, `discount_amount`, `tax_amount`, `shipping_fee`, `total_net_amount`, `created_at`
) VALUES (
    'ord00001-0000-0000-0000-000000000002', 'ORD-2026-0902-002', 
    'u0000001-0000-0000-0000-000000000004', 'a0000001-0000-0000-0000-000000000003', 
    'os000001-0000-0000-0000-000000000001', 'w0000001-0000-0000-0000-000000000002',
    459000.00, 50000.00, 44990.00, 22000.00, 475990.00, '2026-09-02 14:15:00'
);
INSERT INTO `order_items` (`id`, `order_id`, `product_id`, `quantity`, `unit_price`, `subtotal`) VALUES
('oi000001-0000-0000-0000-000000000003', 'ord00001-0000-0000-0000-000000000002', 'p0000001-0000-0000-0000-000000000004', 1, 459000.00, 459000.00);
INSERT INTO `orders` (
    `id`, `order_number`, `customer_id`, `shipping_address_id`, `order_status_id`, `warehouse_id`, 
    `total_gross_amount`, `discount_amount`, `tax_amount`, `shipping_fee`, `total_net_amount`, `created_at`
) VALUES (
    'ord00001-0000-0000-0000-000000000003', 'ORD-2026-0903-003', 
    'u0000001-0000-0000-0000-000000000004', 'a0000001-0000-0000-0000-000000000003', 
    'os000001-0000-0000-0000-000000000002', 'w0000001-0000-0000-0000-000000000002',
    179000.00, 0.00, 19690.00, 12000.00, 210690.00, '2026-09-03 10:00:00'
);
INSERT INTO `order_items` (`id`, `order_id`, `product_id`, `quantity`, `unit_price`, `subtotal`) VALUES
('oi000001-0000-0000-0000-000000000004', 'ord00001-0000-0000-0000-000000000003', 'p0000001-0000-0000-0000-000000000003', 1, 179000.00, 179000.00);
INSERT INTO `orders` (
    `id`, `order_number`, `customer_id`, `shipping_address_id`, `order_status_id`, `warehouse_id`, 
    `total_gross_amount`, `discount_amount`, `tax_amount`, `shipping_fee`, `total_net_amount`, `created_at`
) VALUES (
    'ord00001-0000-0000-0000-000000000004', 'ORD-2026-0904-004', 
    'u0000001-0000-0000-0000-000000000008', 'a0000001-0000-0000-0000-000000000004', 
    'os000001-0000-0000-0000-000000000005', 'w0000001-0000-0000-0000-000000000002',
    68000.00, 0.00, 7480.00, 14000.00, 89480.00, '2026-09-04 11:20:00'
);
INSERT INTO `order_cancellations` (`id`, `order_id`, `cancel_reason`, `cancelled_by_user_id`) VALUES
('cnl00001-0000-0000-0000-000000000001', 'ord00001-0000-0000-0000-000000000004', 'Salah memilih alamat pengiriman, order ulang', 'u0000001-0000-0000-0000-000000000008');
INSERT INTO `order_shipping_details` (`id`, `order_id`, `courier_name`, `tracking_number`, `shipping_cost`) VALUES
('osd00001-0000-0000-0000-000000000001', 'ord00001-0000-0000-0000-000000000001', 'GoSend Instant', 'GOSEND-DPS-883192', 15000.00),
('osd00001-0000-0000-0000-000000000002', 'ord00001-0000-0000-0000-000000000003', 'JNE Reguler', 'JNE-CGK-990112', 12000.00);
INSERT INTO `order_notes` (`id`, `order_id`, `author_user_id`, `note_content`) VALUES
('not00001-0000-0000-0000-000000000001', 'ord00001-0000-0000-0000-000000000001', 'u0000001-0000-0000-0000-000000000003', 'Tolong titip di pos sekuriti jika rumah kosong.');

SET FOREIGN_KEY_CHECKS = 1;
