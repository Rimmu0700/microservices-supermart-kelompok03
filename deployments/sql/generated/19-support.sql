-- File ini DIHASILKAN oleh scripts/split_db.py. Jangan diedit manual.
CREATE DATABASE IF NOT EXISTS `support_db` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE `support_db`;
SET FOREIGN_KEY_CHECKS = 0;

-- ===== Tabel milik support-service =====
CREATE TABLE IF NOT EXISTS `ticket_categories` (
    `id` VARCHAR(36) PRIMARY KEY,
    `category_name` VARCHAR(100) NOT NULL UNIQUE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `ticket_sla_configs` (
    `id` VARCHAR(36) PRIMARY KEY,
    `category_id` VARCHAR(36) NOT NULL,
    `max_response_hours` INT NOT NULL,
    `max_resolution_hours` INT NOT NULL,
    FOREIGN KEY (`category_id`) REFERENCES `ticket_categories`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `customer_tickets` (
    `id` VARCHAR(36) PRIMARY KEY,
    `ticket_code` VARCHAR(64) NOT NULL UNIQUE,
    `customer_id` VARCHAR(36) NOT NULL,
    `order_id` VARCHAR(36) NULL,
    `category_id` VARCHAR(36) NOT NULL,
    `subject` VARCHAR(255) NOT NULL,
    `priority` ENUM('LOW', 'MEDIUM', 'HIGH', 'URGENT') DEFAULT 'MEDIUM',
    `status` ENUM('OPEN', 'IN_PROGRESS', 'RESOLVED', 'CLOSED') DEFAULT 'OPEN',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`category_id`) REFERENCES `ticket_categories`(`id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `ticket_messages` (
    `id` VARCHAR(36) PRIMARY KEY,
    `ticket_id` VARCHAR(36) NOT NULL,
    `sender_user_id` VARCHAR(36) NOT NULL,
    `message_body` TEXT NOT NULL,
    `sent_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`ticket_id`) REFERENCES `customer_tickets`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `ticket_attachments` (
    `id` VARCHAR(36) PRIMARY KEY,
    `ticket_message_id` VARCHAR(36) NOT NULL,
    `attachment_url` VARCHAR(500) NOT NULL,
    FOREIGN KEY (`ticket_message_id`) REFERENCES `ticket_messages`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `ticket_assignments` (
    `id` VARCHAR(36) PRIMARY KEY,
    `ticket_id` VARCHAR(36) NOT NULL,
    `assigned_agent_id` VARCHAR(36) NOT NULL,
    `assigned_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`ticket_id`) REFERENCES `customer_tickets`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `ticket_ratings` (
    `id` VARCHAR(36) PRIMARY KEY,
    `ticket_id` VARCHAR(36) NOT NULL UNIQUE,
    `satisfaction_score` TINYINT NOT NULL CHECK (`satisfaction_score` BETWEEN 1 AND 5),
    `feedback` TEXT NULL,
    FOREIGN KEY (`ticket_id`) REFERENCES `customer_tickets`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `disputes` (
    `id` VARCHAR(36) PRIMARY KEY,
    `dispute_number` VARCHAR(64) NOT NULL UNIQUE,
    `order_id` VARCHAR(36) NOT NULL,
    `claim_amount` DECIMAL(15,2) NOT NULL,
    `status` ENUM('OPEN', 'ARBITRATION', 'RESOLVED') DEFAULT 'OPEN'
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `dispute_evidences` (
    `id` VARCHAR(36) PRIMARY KEY,
    `dispute_id` VARCHAR(36) NOT NULL,
    `evidence_type` VARCHAR(50) NOT NULL,
    `evidence_url` VARCHAR(500) NOT NULL,
    FOREIGN KEY (`dispute_id`) REFERENCES `disputes`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `faq_articles` (
    `id` VARCHAR(36) PRIMARY KEY,
    `category_id` VARCHAR(36) NOT NULL,
    `question` TEXT NOT NULL,
    `answer` TEXT NOT NULL,
    `is_published` TINYINT(1) DEFAULT 1,
    FOREIGN KEY (`category_id`) REFERENCES `ticket_categories`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

-- ===== Replika baca-saja (disinkronkan dari service pemilik, JANGAN ditulis oleh service ini) =====
CREATE TABLE IF NOT EXISTS `users` (
    `id` VARCHAR(36) PRIMARY KEY,
    `email` VARCHAR(255) NOT NULL UNIQUE,
    `full_name` VARCHAR(150) NOT NULL,
    `phone_number` VARCHAR(30) NULL,
    `is_active` TINYINT(1) DEFAULT 1,
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `user_roles` (
    `user_id` VARCHAR(36) NOT NULL,
    `role_id` VARCHAR(36) NOT NULL,
    PRIMARY KEY (`user_id`, `role_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `roles` (
    `id` VARCHAR(36) PRIMARY KEY,
    `role_name` VARCHAR(50) NOT NULL UNIQUE,
    `description` VARCHAR(255) NULL
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
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP
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
INSERT INTO `ticket_categories` (`id`, `category_name`) VALUES
('tc000001-0000-0000-0000-000000000001', 'Kendala Pengiriman & Kurir'),
('tc000001-0000-0000-0000-000000000002', 'Masalah Pembayaran & Tagihan'),
('tc000001-0000-0000-0000-000000000003', 'Kualitas Produk & Retur Barang');
INSERT INTO `ticket_sla_configs` (`id`, `category_id`, `max_response_hours`, `max_resolution_hours`) VALUES
('sla00001-0000-0000-0000-000000000001', 'tc000001-0000-0000-0000-000000000001', 2, 24),
('sla00001-0000-0000-0000-000000000002', 'tc000001-0000-0000-0000-000000000002', 1, 12),
('sla00001-0000-0000-0000-000000000003', 'tc000001-0000-0000-0000-000000000003', 4, 48);
INSERT INTO `customer_tickets` (
    `id`, `ticket_code`, `customer_id`, `order_id`, `category_id`, `subject`, `priority`, `status`, `created_at`
) VALUES (
    'tkt00001-0000-0000-0000-000000000001', 'TKT-202609-001', 
    'u0000001-0000-0000-0000-000000000003', 'ord00001-0000-0000-0000-000000000001', 
    'tc000001-0000-0000-0000-000000000001', 'Konfirmasi Penerimaan Paket Beras', 'LOW', 'RESOLVED', '2026-09-01 12:00:00'
), (
    'tkt00001-0000-0000-0000-000000000002', 'TKT-202609-002', 
    'u0000001-0000-0000-0000-000000000004', 'ord00001-0000-0000-0000-000000000002', 
    'tc000001-0000-0000-0000-000000000002', 'Bantuan Cara Pembayaran Virtual Account', 'MEDIUM', 'OPEN', '2026-09-02 15:00:00'
);
INSERT INTO `ticket_messages` (`id`, `ticket_id`, `sender_user_id`, `message_body`, `sent_at`) VALUES
('tkm00001-0000-0000-0000-000000000001', 'tkt00001-0000-0000-0000-000000000001', 'u0000001-0000-0000-0000-000000000003', 'Halo CS, paket sudah diterima rapi. Terima kasih banyak!', '2026-09-01 12:00:00'),
('tkm00001-0000-0000-0000-000000000002', 'tkt00001-0000-0000-0000-000000000001', 'u0000001-0000-0000-0000-000000000007', 'Sama-sama Bapak Wayan! Senang bisa melayani Anda. Sehat selalu!', '2026-09-01 12:15:00'),
('tkm00001-0000-0000-0000-000000000003', 'tkt00001-0000-0000-0000-000000000002', 'u0000001-0000-0000-0000-000000000004', 'Halo, nomor virtual account BCA saya tidak bisa dicopy, mohon bantuannya.', '2026-09-02 15:00:00');
INSERT INTO `ticket_assignments` (`id`, `ticket_id`, `assigned_agent_id`, `assigned_at`) VALUES
('tas00001-0000-0000-0000-000000000001', 'tkt00001-0000-0000-0000-000000000001', 'u0000001-0000-0000-0000-000000000007', '2026-09-01 12:05:00'),
('tas00001-0000-0000-0000-000000000002', 'tkt00001-0000-0000-0000-000000000002', 'u0000001-0000-0000-0000-000000000007', '2026-09-02 15:05:00');
INSERT INTO `ticket_ratings` (`id`, `ticket_id`, `satisfaction_score`, `feedback`) VALUES
('tr000001-0000-0000-0000-000000000001', 'tkt00001-0000-0000-0000-000000000001', 5, 'Respon CS Rini sangat ramah dan tanggap!');
INSERT INTO `faq_articles` (`id`, `category_id`, `question`, `answer`, `is_published`) VALUES
('faq00001-0000-0000-0000-000000000001', 'tc000001-0000-0000-0000-000000000001', 'Berapa lama estimasi pengiriman instant?', 'Pengiriman instant melalui GoSend tiba dalam 1 hingga 3 jam setelah barang diserahkan ke kurir.', 1),
('faq00001-0000-0000-0000-000000000002', 'tc000001-0000-0000-0000-000000000002', 'Metode pembayaran apa saja yang didukung?', 'Kami mendukung QRIS (Gopay, OVO, ShopeePay), Transfer Virtual Account (BCA, Mandiri), dan Nusantara Store Credit.', 1),
('faq00001-0000-0000-0000-000000000003', 'tc000001-0000-0000-0000-000000000003', 'Bagaimana prosedur retur barang rusak?', 'Laporkan via menu Bantuan & Tiket dalam 1x24 jam sejak barang tiba dengan menyertakan foto unboxing produk.', 1);

SET FOREIGN_KEY_CHECKS = 1;
