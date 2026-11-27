-- File ini DIHASILKAN oleh scripts/split_db.py. Jangan diedit manual.
CREATE DATABASE IF NOT EXISTS `promotion_db` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE `promotion_db`;
SET FOREIGN_KEY_CHECKS = 0;

-- ===== Tabel milik promotion-service =====
CREATE TABLE IF NOT EXISTS `promotions` (
    `id` VARCHAR(36) PRIMARY KEY,
    `promo_code` VARCHAR(50) NOT NULL UNIQUE,
    `promo_name` VARCHAR(150) NOT NULL,
    `start_date` DATETIME NOT NULL,
    `end_date` DATETIME NOT NULL,
    `is_active` TINYINT(1) DEFAULT 1
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `promotion_rules` (
    `id` VARCHAR(36) PRIMARY KEY,
    `promotion_id` VARCHAR(36) NOT NULL,
    `min_purchase_amount` DECIMAL(15,2) DEFAULT 0.00,
    `discount_percentage` DECIMAL(5,2) DEFAULT 0.00,
    `max_discount_cap` DECIMAL(15,2) NULL,
    FOREIGN KEY (`promotion_id`) REFERENCES `promotions`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `discount_tiers` (
    `id` VARCHAR(36) PRIMARY KEY,
    `promotion_id` VARCHAR(36) NOT NULL,
    `tier_level` INT NOT NULL,
    `tier_discount_percent` DECIMAL(5,2) NOT NULL,
    FOREIGN KEY (`promotion_id`) REFERENCES `promotions`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `vouchers` (
    `id` VARCHAR(36) PRIMARY KEY,
    `voucher_code` VARCHAR(50) NOT NULL UNIQUE,
    `voucher_value` DECIMAL(15,2) NOT NULL,
    `quota_limit` INT NOT NULL DEFAULT 100,
    `quota_used` INT NOT NULL DEFAULT 0,
    `expires_at` DATETIME NOT NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `voucher_usages` (
    `id` VARCHAR(36) PRIMARY KEY,
    `voucher_id` VARCHAR(36) NOT NULL,
    `user_id` VARCHAR(36) NOT NULL,
    `order_id` VARCHAR(36) NOT NULL,
    `used_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`voucher_id`) REFERENCES `vouchers`(`id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `voucher_customer_eligibilities` (
    `voucher_id` VARCHAR(36) NOT NULL,
    `user_id` VARCHAR(36) NOT NULL,
    PRIMARY KEY (`voucher_id`, `user_id`),
    FOREIGN KEY (`voucher_id`) REFERENCES `vouchers`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `coupons` (
    `id` VARCHAR(36) PRIMARY KEY,
    `coupon_serial` VARCHAR(64) NOT NULL UNIQUE,
    `is_redeemed` TINYINT(1) DEFAULT 0
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `bundle_packages` (
    `id` VARCHAR(36) PRIMARY KEY,
    `package_name` VARCHAR(150) NOT NULL,
    `package_price` DECIMAL(15,2) NOT NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `bundle_items` (
    `bundle_id` VARCHAR(36) NOT NULL,
    `product_id` VARCHAR(36) NOT NULL,
    `quantity` INT NOT NULL DEFAULT 1,
    PRIMARY KEY (`bundle_id`, `product_id`),
    FOREIGN KEY (`bundle_id`) REFERENCES `bundle_packages`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `loyalty_points` (
    `id` VARCHAR(36) PRIMARY KEY,
    `user_id` VARCHAR(36) NOT NULL UNIQUE,
    `current_points` INT DEFAULT 0
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `point_transactions` (
    `id` VARCHAR(36) PRIMARY KEY,
    `loyalty_point_id` VARCHAR(36) NOT NULL,
    `points` INT NOT NULL,
    `point_type` ENUM('EARNED', 'REDEEMED') NOT NULL,
    `reference_order_id` VARCHAR(36) NULL,
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`loyalty_point_id`) REFERENCES `loyalty_points`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `point_redemptions` (
    `id` VARCHAR(36) PRIMARY KEY,
    `user_id` VARCHAR(36) NOT NULL,
    `reward_name` VARCHAR(150) NOT NULL,
    `points_deducted` INT NOT NULL,
    `redeemed_at` DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `cashback_programs` (
    `id` VARCHAR(36) PRIMARY KEY,
    `program_name` VARCHAR(150) NOT NULL,
    `cashback_percentage` DECIMAL(5,2) NOT NULL,
    `max_cashback_amount` DECIMAL(15,2) NOT NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `flash_sales` (
    `id` VARCHAR(36) PRIMARY KEY,
    `product_id` VARCHAR(36) NOT NULL,
    `flash_price` DECIMAL(15,2) NOT NULL,
    `allocated_stock` INT NOT NULL,
    `starts_at` DATETIME NOT NULL,
    `ends_at` DATETIME NOT NULL
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
INSERT INTO `promotions` (`id`, `promo_code`, `promo_name`, `start_date`, `end_date`, `is_active`) VALUES
('pro00001-0000-0000-0000-000000000001', 'SEMBARANG-SEPT', 'Promo Merdeka Belanja Sembako', '2026-09-01 00:00:00', '2026-09-30 23:59:59', 1),
('pro00001-0000-0000-0000-000000000002', 'TECHFEST-2026', 'Festival Perangkat Kerja & Rumah', '2026-09-01 00:00:00', '2026-10-15 23:59:59', 1);
INSERT INTO `promotion_rules` (`id`, `promotion_id`, `min_purchase_amount`, `discount_percentage`, `max_discount_cap`) VALUES
('pr000001-0000-0000-0000-000000000001', 'pro00001-0000-0000-0000-000000000001', 100000.00, 10.00, 25000.00),
('pr000001-0000-0000-0000-000000000002', 'pro00001-0000-0000-0000-000000000002', 300000.00, 15.00, 50000.00);
INSERT INTO `vouchers` (`id`, `voucher_code`, `voucher_value`, `quota_limit`, `quota_used`, `expires_at`) VALUES
('v0000001-0000-0000-0000-000000000001', 'BALISEHAT20', 20000.00, 500, 48, '2027-12-31 23:59:59'),
('v0000001-0000-0000-0000-000000000002', 'TECHFEST50', 50000.00, 100, 12, '2027-12-31 23:59:59'),
('v0000001-0000-0000-0000-000000000003', 'GRATISONGKIR', 15000.00, 1000, 120, '2027-12-31 23:59:59'),
('v0000001-0000-0000-0000-000000000004', 'WELCOME10', 10000.00, 2000, 85, '2027-12-31 23:59:59');
INSERT INTO `voucher_usages` (`id`, `voucher_id`, `user_id`, `order_id`) VALUES
('vu000001-0000-0000-0000-000000000001', 'v0000001-0000-0000-0000-000000000001', 'u0000001-0000-0000-0000-000000000003', 'ord00001-0000-0000-0000-000000000001');
INSERT INTO `loyalty_points` (`id`, `user_id`, `current_points`) VALUES
('loy00001-0000-0000-0000-000000000001', 'u0000001-0000-0000-0000-000000000003', 280),
('loy00001-0000-0000-0000-000000000002', 'u0000001-0000-0000-0000-000000000004', 520);
INSERT INTO `point_transactions` (`id`, `loyalty_point_id`, `points`, `point_type`, `reference_order_id`) VALUES
('ptx00001-0000-0000-0000-000000000001', 'loy00001-0000-0000-0000-000000000001', 280, 'EARNED', 'ord00001-0000-0000-0000-000000000001');
INSERT INTO `flash_sales` (`id`, `product_id`, `flash_price`, `allocated_stock`, `starts_at`, `ends_at`) VALUES
('fs000001-0000-0000-0000-000000000001', 'p0000001-0000-0000-0000-000000000003', 149000.00, 30, '2026-09-05 12:00:00', '2026-09-10 18:00:00');

SET FOREIGN_KEY_CHECKS = 1;
