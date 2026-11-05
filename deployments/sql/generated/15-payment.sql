-- File ini DIHASILKAN oleh scripts/split_db.py. Jangan diedit manual.
CREATE DATABASE IF NOT EXISTS `payment_db` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE `payment_db`;
SET FOREIGN_KEY_CHECKS = 0;

-- ===== Tabel milik payment-service =====
CREATE TABLE IF NOT EXISTS `payment_methods` (
    `id` VARCHAR(36) PRIMARY KEY,
    `method_code` VARCHAR(50) NOT NULL UNIQUE,
    `method_name` VARCHAR(100) NOT NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `payment_gateways` (
    `id` VARCHAR(36) PRIMARY KEY,
    `gateway_name` VARCHAR(100) NOT NULL,
    `api_endpoint` VARCHAR(255) NOT NULL,
    `is_active` TINYINT(1) DEFAULT 1
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `payment_invoices` (
    `id` VARCHAR(36) PRIMARY KEY,
    `invoice_number` VARCHAR(64) NOT NULL UNIQUE,
    `order_id` VARCHAR(36) NOT NULL,
    `customer_id` VARCHAR(36) NOT NULL,
    `amount` DECIMAL(15,2) NOT NULL,
    `payment_status` ENUM('UNPAID', 'PAID', 'EXPIRED', 'CANCELLED') DEFAULT 'UNPAID',
    `due_date` DATETIME NOT NULL,
    `paid_at` DATETIME NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `payment_transactions` (
    `id` VARCHAR(36) PRIMARY KEY,
    `invoice_id` VARCHAR(36) NOT NULL,
    `payment_method_id` VARCHAR(36) NOT NULL,
    `gateway_id` VARCHAR(36) NULL,
    `transaction_reference` VARCHAR(100) NOT NULL UNIQUE,
    `amount_paid` DECIMAL(15,2) NOT NULL,
    `status` ENUM('PENDING', 'SUCCESS', 'FAILED') DEFAULT 'PENDING',
    `transaction_time` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`invoice_id`) REFERENCES `payment_invoices`(`id`) ON DELETE CASCADE,
    FOREIGN KEY (`payment_method_id`) REFERENCES `payment_methods`(`id`),
    FOREIGN KEY (`gateway_id`) REFERENCES `payment_gateways`(`id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `payment_callbacks` (
    `id` VARCHAR(36) PRIMARY KEY,
    `gateway_id` VARCHAR(36) NOT NULL,
    `payload` JSON NOT NULL,
    `received_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`gateway_id`) REFERENCES `payment_gateways`(`id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `refunds` (
    `id` VARCHAR(36) PRIMARY KEY,
    `refund_number` VARCHAR(64) NOT NULL UNIQUE,
    `order_id` VARCHAR(36) NOT NULL,
    `requested_by_user_id` VARCHAR(36) NOT NULL,
    `total_refund_amount` DECIMAL(15,2) NOT NULL,
    `refund_status` ENUM('PENDING', 'APPROVED', 'REJECTED', 'COMPLETED') DEFAULT 'PENDING',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `refund_items` (
    `id` VARCHAR(36) PRIMARY KEY,
    `refund_id` VARCHAR(36) NOT NULL,
    `order_item_id` VARCHAR(36) NOT NULL,
    `quantity` INT NOT NULL,
    `refund_subtotal` DECIMAL(15,2) NOT NULL,
    FOREIGN KEY (`refund_id`) REFERENCES `refunds`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `refund_transactions` (
    `id` VARCHAR(36) PRIMARY KEY,
    `refund_id` VARCHAR(36) NOT NULL,
    `transaction_reference` VARCHAR(100) NOT NULL,
    `disbursed_amount` DECIMAL(15,2) NOT NULL,
    `disbursed_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`refund_id`) REFERENCES `refunds`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `escrow_accounts` (
    `id` VARCHAR(36) PRIMARY KEY,
    `order_id` VARCHAR(36) NOT NULL UNIQUE,
    `held_amount` DECIMAL(15,2) NOT NULL,
    `is_released` TINYINT(1) DEFAULT 0
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `store_credits` (
    `id` VARCHAR(36) PRIMARY KEY,
    `user_id` VARCHAR(36) NOT NULL UNIQUE,
    `balance` DECIMAL(15,2) DEFAULT 0.00
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `credit_transactions` (
    `id` VARCHAR(36) PRIMARY KEY,
    `store_credit_id` VARCHAR(36) NOT NULL,
    `amount` DECIMAL(15,2) NOT NULL,
    `transaction_type` ENUM('CREDIT', 'DEBIT') NOT NULL,
    `description` VARCHAR(255) NULL,
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`store_credit_id`) REFERENCES `store_credits`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `disbursement_ledgers` (
    `id` VARCHAR(36) PRIMARY KEY,
    `disbursement_reference` VARCHAR(64) NOT NULL UNIQUE,
    `amount` DECIMAL(15,2) NOT NULL,
    `recipient_bank_code` VARCHAR(30) NOT NULL,
    `recipient_account_number` VARCHAR(50) NOT NULL,
    `status` ENUM('PENDING', 'PROCESSED', 'FAILED') DEFAULT 'PENDING'
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
INSERT INTO `payment_gateways` (`id`, `gateway_name`, `api_endpoint`, `is_active`) VALUES
('gw000001-0000-0000-0000-000000000001', 'Midtrans Snap Simulator', 'https://app.sandbox.midtrans.com/snap/v1', 1),
('gw000001-0000-0000-0000-000000000002', 'Xendit Invoice Simulator', 'https://api.xendit.co/v2/invoices', 1);
INSERT INTO `payment_methods` (`id`, `method_code`, `method_name`) VALUES
('pm000001-0000-0000-0000-000000000001', 'QRIS', 'QRIS Dinamis (Gopay/OVO/ShopeePay/BCA)'),
('pm000001-0000-0000-0000-000000000002', 'BCA_VA', 'BCA Virtual Account'),
('pm000001-0000-0000-0000-000000000003', 'MANDIRI_VA', 'Mandiri Virtual Account'),
('pm000001-0000-0000-0000-000000000004', 'STORE_CREDIT', 'Nusantara Store Credit / Dompet');
INSERT INTO `payment_invoices` (`id`, `invoice_number`, `order_id`, `customer_id`, `amount`, `payment_status`, `due_date`, `paid_at`) VALUES
('inv00001-0000-0000-0000-000000000001', 'INV-20260901-001', 'ord00001-0000-0000-0000-000000000001', 'u0000001-0000-0000-0000-000000000003', 279180.00, 'PAID', '2026-09-02 09:30:00', '2026-09-01 09:35:12'),
('inv00001-0000-0000-0000-000000000002', 'INV-20260902-002', 'ord00001-0000-0000-0000-000000000002', 'u0000001-0000-0000-0000-000000000004', 475990.00, 'UNPAID', '2026-09-03 14:15:00', NULL),
('inv00001-0000-0000-0000-000000000003', 'INV-20260903-003', 'ord00001-0000-0000-0000-000000000003', 'u0000001-0000-0000-0000-000000000004', 210690.00, 'PAID', '2026-09-04 10:00:00', '2026-09-03 10:05:30');
INSERT INTO `payment_transactions` (`id`, `invoice_id`, `payment_method_id`, `gateway_id`, `transaction_reference`, `amount_paid`, `status`, `transaction_time`) VALUES
('ptx00001-0000-0000-0000-000000000001', 'inv00001-0000-0000-0000-000000000001', 'pm000001-0000-0000-0000-000000000001', 'gw000001-0000-0000-0000-000000000001', 'TRX-QRIS-992817263', 279180.00, 'SUCCESS', '2026-09-01 09:35:12'),
('ptx00001-0000-0000-0000-000000000002', 'inv00001-0000-0000-0000-000000000003', 'pm000001-0000-0000-0000-000000000002', 'gw000001-0000-0000-0000-000000000002', 'TRX-BCA-881928371', 210690.00, 'SUCCESS', '2026-09-03 10:05:30');
INSERT INTO `store_credits` (`id`, `user_id`, `balance`) VALUES
('sc000001-0000-0000-0000-000000000001', 'u0000001-0000-0000-0000-000000000003', 150000.00),
('sc000001-0000-0000-0000-000000000002', 'u0000001-0000-0000-0000-000000000004', 50000.00),
('sc000001-0000-0000-0000-000000000003', 'u0000001-0000-0000-0000-000000000008', 25000.00);
INSERT INTO `credit_transactions` (`id`, `store_credit_id`, `amount`, `transaction_type`, `description`) VALUES
('ctx00001-0000-0000-0000-000000000001', 'sc000001-0000-0000-0000-000000000001', 150000.00, 'CREDIT', 'Topup saldo dompet retail');

SET FOREIGN_KEY_CHECKS = 1;
