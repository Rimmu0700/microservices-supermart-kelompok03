-- File ini DIHASILKAN oleh scripts/split_db.py. Jangan diedit manual.
CREATE DATABASE IF NOT EXISTS `auth_db` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE `auth_db`;
SET FOREIGN_KEY_CHECKS = 0;

-- ===== Tabel milik auth-service =====
CREATE TABLE IF NOT EXISTS `users` (
    `id` VARCHAR(36) PRIMARY KEY,
    `email` VARCHAR(255) NOT NULL UNIQUE,
    `password_hash` VARCHAR(255) NOT NULL,
    `full_name` VARCHAR(150) NOT NULL,
    `phone_number` VARCHAR(30) NULL,
    `is_active` TINYINT(1) DEFAULT 1,
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `user_profiles` (
    `id` VARCHAR(36) PRIMARY KEY,
    `user_id` VARCHAR(36) NOT NULL UNIQUE,
    `gender` ENUM('MALE', 'FEMALE', 'OTHER') NULL,
    `birth_date` DATE NULL,
    `avatar_url` VARCHAR(500) NULL,
    `bio` TEXT NULL,
    FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
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
    `is_primary` TINYINT(1) DEFAULT 0,
    FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `user_kyc_documents` (
    `id` VARCHAR(36) PRIMARY KEY,
    `user_id` VARCHAR(36) NOT NULL,
    `id_card_number` VARCHAR(50) NOT NULL,
    `id_card_image_url` VARCHAR(500) NOT NULL,
    `selfie_image_url` VARCHAR(500) NOT NULL,
    `verification_status` ENUM('PENDING', 'VERIFIED', 'REJECTED') DEFAULT 'PENDING',
    `verified_at` DATETIME NULL,
    FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `roles` (
    `id` VARCHAR(36) PRIMARY KEY,
    `role_name` VARCHAR(50) NOT NULL UNIQUE,
    `description` VARCHAR(255) NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `permissions` (
    `id` VARCHAR(36) PRIMARY KEY,
    `permission_key` VARCHAR(100) NOT NULL UNIQUE,
    `description` VARCHAR(255) NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `role_permissions` (
    `role_id` VARCHAR(36) NOT NULL,
    `permission_id` VARCHAR(36) NOT NULL,
    PRIMARY KEY (`role_id`, `permission_id`),
    FOREIGN KEY (`role_id`) REFERENCES `roles`(`id`) ON DELETE CASCADE,
    FOREIGN KEY (`permission_id`) REFERENCES `permissions`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `user_roles` (
    `user_id` VARCHAR(36) NOT NULL,
    `role_id` VARCHAR(36) NOT NULL,
    PRIMARY KEY (`user_id`, `role_id`),
    FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE,
    FOREIGN KEY (`role_id`) REFERENCES `roles`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `auth_tokens` (
    `id` VARCHAR(36) PRIMARY KEY,
    `user_id` VARCHAR(36) NOT NULL,
    `token_value` VARCHAR(500) NOT NULL UNIQUE,
    `token_type` ENUM('ACCESS', 'REFRESH') NOT NULL,
    `expires_at` DATETIME NOT NULL,
    FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `login_histories` (
    `id` VARCHAR(36) PRIMARY KEY,
    `user_id` VARCHAR(36) NOT NULL,
    `ip_address` VARCHAR(45) NOT NULL,
    `user_agent` TEXT NULL,
    `login_time` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `password_resets` (
    `id` VARCHAR(36) PRIMARY KEY,
    `user_id` VARCHAR(36) NOT NULL,
    `reset_token` VARCHAR(255) NOT NULL UNIQUE,
    `is_used` TINYINT(1) DEFAULT 0,
    `expires_at` DATETIME NOT NULL,
    FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `two_factor_auths` (
    `id` VARCHAR(36) PRIMARY KEY,
    `user_id` VARCHAR(36) NOT NULL UNIQUE,
    `secret_key` VARCHAR(255) NOT NULL,
    `is_enabled` TINYINT(1) DEFAULT 0,
    FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
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
INSERT INTO `roles` (`id`, `role_name`, `description`) VALUES
('r0000001-0000-0000-0000-000000000001', 'SUPER_ADMIN', 'Akses penuh ke seluruh sistem operasional'),
('r0000001-0000-0000-0000-000000000002', 'WAREHOUSE_STAFF', 'Operator stok dan pemrosesan barang di gudang'),
('r0000001-0000-0000-0000-000000000003', 'CUSTOMER', 'Pelanggan retail umum'),
('r0000001-0000-0000-0000-000000000004', 'COURIER', 'Pengemudi logistik dan kurir pengiriman'),
('r0000001-0000-0000-0000-000000000005', 'CS_AGENT', 'Layanan pelanggan dan resolusi tiket kendala');
INSERT INTO `permissions` (`id`, `permission_key`, `description`) VALUES
('pm000001-0000-0000-0000-000000000001', 'users.manage', 'Kelola data pengguna dan perizinan'),
('pm000001-0000-0000-0000-000000000002', 'catalog.manage', 'Kelola produk, kategori, dan brand'),
('pm000001-0000-0000-0000-000000000003', 'inventory.manage', 'Kelola stok dan mutasi gudang'),
('pm000001-0000-0000-0000-000000000004', 'orders.manage', 'Kelola dan perbarui status pesanan'),
('pm000001-0000-0000-0000-000000000005', 'payments.manage', 'Kelola tagihan, faktur, dan verifikasi refund'),
('pm000001-0000-0000-0000-000000000006', 'promotions.manage', 'Kelola program promosi, diskon, dan voucher'),
('pm000001-0000-0000-0000-000000000007', 'logistics.manage', 'Kelola armada pengiriman, kurir, dan rute'),
('pm000001-0000-0000-0000-000000000008', 'procurement.manage', 'Kelola purchase order pemasok dan penerimaan GRN'),
('pm000001-0000-0000-0000-000000000009', 'support.manage', 'Tangani tiket keluhan pelanggan dan FAQ');
INSERT INTO `role_permissions` (`role_id`, `permission_id`) VALUES
('r0000001-0000-0000-0000-000000000001', 'pm000001-0000-0000-0000-000000000001'),
('r0000001-0000-0000-0000-000000000001', 'pm000001-0000-0000-0000-000000000002'),
('r0000001-0000-0000-0000-000000000001', 'pm000001-0000-0000-0000-000000000003'),
('r0000001-0000-0000-0000-000000000001', 'pm000001-0000-0000-0000-000000000004'),
('r0000001-0000-0000-0000-000000000001', 'pm000001-0000-0000-0000-000000000005'),
('r0000001-0000-0000-0000-000000000001', 'pm000001-0000-0000-0000-000000000006'),
('r0000001-0000-0000-0000-000000000001', 'pm000001-0000-0000-0000-000000000007'),
('r0000001-0000-0000-0000-000000000001', 'pm000001-0000-0000-0000-000000000008'),
('r0000001-0000-0000-0000-000000000001', 'pm000001-0000-0000-0000-000000000009'),
('r0000001-0000-0000-0000-000000000002', 'pm000001-0000-0000-0000-000000000003'),
('r0000001-0000-0000-0000-000000000002', 'pm000001-0000-0000-0000-000000000008'),
('r0000001-0000-0000-0000-000000000004', 'pm000001-0000-0000-0000-000000000007'),
('r0000001-0000-0000-0000-000000000005', 'pm000001-0000-0000-0000-000000000009');
INSERT INTO `users` (`id`, `email`, `password_hash`, `full_name`, `phone_number`, `is_active`) VALUES
('u0000001-0000-0000-0000-000000000001', 'admin@nusantara.co.id', '$2a$10$UwQm.bwc/69L3mHOIx89VOgaeQhi9cJQvtTm.Y5/AbW2P7NRqgfBC', 'Budi Hartono (Admin)', '081122334455', 1),
('u0000001-0000-0000-0000-000000000002', 'gudang.jakarta@nusantara.co.id', '$2a$10$UwQm.bwc/69L3mHOIx89VOgaeQhi9cJQvtTm.Y5/AbW2P7NRqgfBC', 'Siti Rahma (Staff Gudang JKT)', '081299887766', 1),
('u0000001-0000-0000-0000-000000000003', 'wayan.putra@gmail.com', '$2a$10$UwQm.bwc/69L3mHOIx89VOgaeQhi9cJQvtTm.Y5/AbW2P7NRqgfBC', 'I Wayan Putra Adnyana', '081338001122', 1),
('u0000001-0000-0000-0000-000000000004', 'dewi.lestari@yahoo.com', '$2a$10$UwQm.bwc/69L3mHOIx89VOgaeQhi9cJQvtTm.Y5/AbW2P7NRqgfBC', 'Ni Made Dewi Lestari', '081999112233', 1),
('u0000001-0000-0000-0000-000000000005', 'gudang.denpasar@nusantara.co.id', '$2a$10$UwQm.bwc/69L3mHOIx89VOgaeQhi9cJQvtTm.Y5/AbW2P7NRqgfBC', 'Made Suartana (Staff Gudang DPS)', '081338776655', 1),
('u0000001-0000-0000-0000-000000000006', 'kurir.anto@nusantara.co.id', '$2a$10$UwQm.bwc/69L3mHOIx89VOgaeQhi9cJQvtTm.Y5/AbW2P7NRqgfBC', 'Anto Pratama (Kurir Express)', '081888223344', 1),
('u0000001-0000-0000-0000-000000000007', 'cs.agent@nusantara.co.id', '$2a$10$UwQm.bwc/69L3mHOIx89VOgaeQhi9cJQvtTm.Y5/AbW2P7NRqgfBC', 'Rini Indah (CS Support Specialist)', '081777334455', 1),
('u0000001-0000-0000-0000-000000000008', 'budi.santoso@gmail.com', '$2a$10$UwQm.bwc/69L3mHOIx89VOgaeQhi9cJQvtTm.Y5/AbW2P7NRqgfBC', 'Budi Santoso (Customer)', '081234567890', 1);
INSERT INTO `user_roles` (`user_id`, `role_id`) VALUES
('u0000001-0000-0000-0000-000000000001', 'r0000001-0000-0000-0000-000000000001'),
('u0000001-0000-0000-0000-000000000002', 'r0000001-0000-0000-0000-000000000002'),
('u0000001-0000-0000-0000-000000000003', 'r0000001-0000-0000-0000-000000000003'),
('u0000001-0000-0000-0000-000000000004', 'r0000001-0000-0000-0000-000000000003'),
('u0000001-0000-0000-0000-000000000005', 'r0000001-0000-0000-0000-000000000002'),
('u0000001-0000-0000-0000-000000000006', 'r0000001-0000-0000-0000-000000000004'),
('u0000001-0000-0000-0000-000000000007', 'r0000001-0000-0000-0000-000000000005'),
('u0000001-0000-0000-0000-000000000008', 'r0000001-0000-0000-0000-000000000003');
INSERT INTO `user_profiles` (`id`, `user_id`, `gender`, `birth_date`, `avatar_url`, `bio`) VALUES
('prof0001-0000-0000-0000-000000000001', 'u0000001-0000-0000-0000-000000000001', 'MALE', '1985-05-12', 'https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=200', 'Super Administrator Sistem PT Nusantara SuperMart'),
('prof0001-0000-0000-0000-000000000002', 'u0000001-0000-0000-0000-000000000002', 'FEMALE', '1992-08-20', 'https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=200', 'Warehouse Lead Cengkareng DC'),
('prof0001-0000-0000-0000-000000000003', 'u0000001-0000-0000-0000-000000000003', 'MALE', '1995-11-04', 'https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=200', 'Loyal SuperMart member from Denpasar'),
('prof0001-0000-0000-0000-000000000004', 'u0000001-0000-0000-0000-000000000004', 'FEMALE', '1998-03-15', 'https://images.unsplash.com/photo-1438761681033-6461ffad8d80?w=200', 'Tech enthusiast and coffee lover from Jakarta');
INSERT INTO `user_addresses` (`id`, `user_id`, `address_label`, `recipient_name`, `phone_number`, `street_address`, `city`, `province`, `postal_code`, `latitude`, `longitude`, `is_primary`) VALUES
('a0000001-0000-0000-0000-000000000001', 'u0000001-0000-0000-0000-000000000003', 'Rumah Denpasar', 'Wayan Putra', '081338001122', 'Jl. Hayam Wuruk No. 88, Tanjung Bungkak', 'Denpasar', 'Bali', '80239', -8.6539, 115.2341, 1),
('a0000001-0000-0000-0000-000000000002', 'u0000001-0000-0000-0000-000000000003', 'Kantor Renon', 'Wayan Putra (Office)', '081338001122', 'Jl. Raya Puputan No. 12, Renon', 'Denpasar', 'Bali', '80234', -8.6705, 115.2335, 0),
('a0000001-0000-0000-0000-000000000003', 'u0000001-0000-0000-0000-000000000004', 'Apartemen Sudirman', 'Dewi Lestari', '081999112233', 'Jl. Jend. Sudirman Kav. 45, Tower A Lt. 12', 'Jakarta Selatan', 'DKI Jakarta', '12190', -6.2154, 106.8185, 1),
('a0000001-0000-0000-0000-000000000004', 'u0000001-0000-0000-0000-000000000008', 'Rumah Bandung', 'Budi Santoso', '081234567890', 'Jl. Dago No. 150, Coblong', 'Bandung', 'Jawa Barat', '40135', -6.8856, 107.6133, 1);
INSERT INTO `user_kyc_documents` (`id`, `user_id`, `id_card_number`, `id_card_image_url`, `selfie_image_url`, `verification_status`, `verified_at`) VALUES
('kyc00001-0000-0000-0000-000000000001', 'u0000001-0000-0000-0000-000000000003', '5171010411950001', 'https://cdn.nusantara.com/kyc/ktp_wayan.jpg', 'https://cdn.nusantara.com/kyc/selfie_wayan.jpg', 'VERIFIED', '2026-08-15 10:00:00'),
('kyc00001-0000-0000-0000-000000000002', 'u0000001-0000-0000-0000-000000000004', '3174025503980002', 'https://cdn.nusantara.com/kyc/ktp_dewi.jpg', 'https://cdn.nusantara.com/kyc/selfie_dewi.jpg', 'VERIFIED', '2026-08-20 14:30:00');

SET FOREIGN_KEY_CHECKS = 1;
