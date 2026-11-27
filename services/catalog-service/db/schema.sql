-- File ini DIHASILKAN oleh scripts/split_db.py. Jangan diedit manual.
CREATE DATABASE IF NOT EXISTS `catalog_db` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE `catalog_db`;
SET FOREIGN_KEY_CHECKS = 0;

-- ===== Tabel milik catalog-service =====
CREATE TABLE IF NOT EXISTS `brands` (
    `id` VARCHAR(36) PRIMARY KEY,
    `brand_name` VARCHAR(100) NOT NULL UNIQUE,
    `logo_url` VARCHAR(500) NULL,
    `website_url` VARCHAR(255) NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `categories` (
    `id` VARCHAR(36) PRIMARY KEY,
    `category_name` VARCHAR(100) NOT NULL,
    `slug` VARCHAR(150) NOT NULL UNIQUE,
    `icon_url` VARCHAR(500) NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `category_hierarchies` (
    `parent_category_id` VARCHAR(36) NOT NULL,
    `child_category_id` VARCHAR(36) NOT NULL,
    `depth_level` INT DEFAULT 1,
    PRIMARY KEY (`parent_category_id`, `child_category_id`),
    FOREIGN KEY (`parent_category_id`) REFERENCES `categories`(`id`) ON DELETE CASCADE,
    FOREIGN KEY (`child_category_id`) REFERENCES `categories`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `products` (
    `id` VARCHAR(36) PRIMARY KEY,
    `sku` VARCHAR(64) NOT NULL UNIQUE,
    `title` VARCHAR(255) NOT NULL,
    `description` TEXT NULL,
    `base_price` DECIMAL(15,2) NOT NULL,
    `brand_id` VARCHAR(36) NULL,
    `weight_gram` INT NOT NULL DEFAULT 0,
    `is_published` TINYINT(1) DEFAULT 1,
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`brand_id`) REFERENCES `brands`(`id`) ON DELETE SET NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `product_translations` (
    `id` VARCHAR(36) PRIMARY KEY,
    `product_id` VARCHAR(36) NOT NULL,
    `language_code` VARCHAR(10) NOT NULL,
    `translated_title` VARCHAR(255) NOT NULL,
    `translated_description` TEXT NULL,
    FOREIGN KEY (`product_id`) REFERENCES `products`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `product_categories` (
    `product_id` VARCHAR(36) NOT NULL,
    `category_id` VARCHAR(36) NOT NULL,
    PRIMARY KEY (`product_id`, `category_id`),
    FOREIGN KEY (`product_id`) REFERENCES `products`(`id`) ON DELETE CASCADE,
    FOREIGN KEY (`category_id`) REFERENCES `categories`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `product_attributes` (
    `id` VARCHAR(36) PRIMARY KEY,
    `attribute_name` VARCHAR(100) NOT NULL UNIQUE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `attribute_values` (
    `id` VARCHAR(36) PRIMARY KEY,
    `attribute_id` VARCHAR(36) NOT NULL,
    `value_name` VARCHAR(100) NOT NULL,
    FOREIGN KEY (`attribute_id`) REFERENCES `product_attributes`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `product_attribute_values` (
    `product_id` VARCHAR(36) NOT NULL,
    `attribute_value_id` VARCHAR(36) NOT NULL,
    PRIMARY KEY (`product_id`, `attribute_value_id`),
    FOREIGN KEY (`product_id`) REFERENCES `products`(`id`) ON DELETE CASCADE,
    FOREIGN KEY (`attribute_value_id`) REFERENCES `attribute_values`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `product_images` (
    `id` VARCHAR(36) PRIMARY KEY,
    `product_id` VARCHAR(36) NOT NULL,
    `image_url` VARCHAR(500) NOT NULL,
    `is_primary` TINYINT(1) DEFAULT 0,
    `display_order` INT DEFAULT 0,
    FOREIGN KEY (`product_id`) REFERENCES `products`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `product_variants` (
    `id` VARCHAR(36) PRIMARY KEY,
    `product_id` VARCHAR(36) NOT NULL,
    `variant_sku` VARCHAR(64) NOT NULL UNIQUE,
    `additional_price` DECIMAL(15,2) DEFAULT 0.00,
    FOREIGN KEY (`product_id`) REFERENCES `products`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `product_variant_options` (
    `variant_id` VARCHAR(36) NOT NULL,
    `attribute_value_id` VARCHAR(36) NOT NULL,
    PRIMARY KEY (`variant_id`, `attribute_value_id`),
    FOREIGN KEY (`variant_id`) REFERENCES `product_variants`(`id`) ON DELETE CASCADE,
    FOREIGN KEY (`attribute_value_id`) REFERENCES `attribute_values`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `tags` (
    `id` VARCHAR(36) PRIMARY KEY,
    `tag_name` VARCHAR(50) NOT NULL UNIQUE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `product_tags` (
    `product_id` VARCHAR(36) NOT NULL,
    `tag_id` VARCHAR(36) NOT NULL,
    PRIMARY KEY (`product_id`, `tag_id`),
    FOREIGN KEY (`product_id`) REFERENCES `products`(`id`) ON DELETE CASCADE,
    FOREIGN KEY (`tag_id`) REFERENCES `tags`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `product_barcodes` (
    `id` VARCHAR(36) PRIMARY KEY,
    `product_id` VARCHAR(36) NOT NULL,
    `barcode_number` VARCHAR(100) NOT NULL UNIQUE,
    `barcode_type` VARCHAR(30) DEFAULT 'EAN-13',
    FOREIGN KEY (`product_id`) REFERENCES `products`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `product_reviews` (
    `id` VARCHAR(36) PRIMARY KEY,
    `product_id` VARCHAR(36) NOT NULL,
    `user_id` VARCHAR(36) NOT NULL,
    `rating` TINYINT NOT NULL CHECK (`rating` BETWEEN 1 AND 5),
    `review_text` TEXT NULL,
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`product_id`) REFERENCES `products`(`id`) ON DELETE CASCADE
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
INSERT INTO `brands` (`id`, `brand_name`, `logo_url`, `website_url`) VALUES
('b0000001-0000-0000-0000-000000000001', 'Indofood Agro', 'https://cdn.nusantara.com/brands/indofood.png', 'https://indofood.com'),
('b0000001-0000-0000-0000-000000000002', 'Kintamani Coffee Roastery', 'https://cdn.nusantara.com/brands/kintamani.png', 'https://kintamanicoffee.id'),
('b0000001-0000-0000-0000-000000000003', 'Logitech Indonesia', 'https://cdn.nusantara.com/brands/logitech.png', 'https://logitech.com'),
('b0000001-0000-0000-0000-000000000004', 'Nutrilon Royal', 'https://cdn.nusantara.com/brands/nutrilon.png', 'https://nutriclub.co.id'),
('b0000001-0000-0000-0000-000000000005', 'Unilever Fresh Mart', 'https://cdn.nusantara.com/brands/unilever.png', 'https://unilever.co.id'),
('b0000001-0000-0000-0000-000000000006', 'Samsung Electronics ID', 'https://cdn.nusantara.com/brands/samsung.png', 'https://samsung.com/id');
INSERT INTO `categories` (`id`, `category_name`, `slug`, `icon_url`) VALUES
('c0000001-0000-0000-0000-000000000001', 'Bahan Pokok & Sembako', 'bahan-pokok-sembako', 'https://cdn.nusantara.com/cat/sembako.png'),
('c0000001-0000-0000-0000-000000000002', 'Kopi & Minuman Organik', 'kopi-minuman-organik', 'https://cdn.nusantara.com/cat/beverage.png'),
('c0000001-0000-0000-0000-000000000003', 'Aksesoris Komputer & Gadget', 'aksesoris-komputer', 'https://cdn.nusantara.com/cat/computer.png'),
('c0000001-0000-0000-0000-000000000004', 'Ibu & Bayi', 'ibu-dan-bayi', 'https://cdn.nusantara.com/cat/baby.png'),
('c0000001-0000-0000-0000-000000000005', 'Perawatan Rumah & Dapur', 'perawatan-rumah', 'https://cdn.nusantara.com/cat/home.png'),
('c0000001-0000-0000-0000-000000000006', 'Elektronik & Smart Home', 'elektronik-smart-home', 'https://cdn.nusantara.com/cat/electronics.png'),
('c0000001-0000-0000-0000-000000000007', 'Beras & Padi Pilihan', 'beras-dan-padi', 'https://cdn.nusantara.com/cat/rice.png'),
('c0000001-0000-0000-0000-000000000008', 'Kopi Spesialti Nusantara', 'kopi-spesialti', 'https://cdn.nusantara.com/cat/coffee.png');
INSERT INTO `category_hierarchies` (`parent_category_id`, `child_category_id`, `depth_level`) VALUES
('c0000001-0000-0000-0000-000000000001', 'c0000001-0000-0000-0000-000000000007', 1),
('c0000001-0000-0000-0000-000000000002', 'c0000001-0000-0000-0000-000000000008', 1);
INSERT INTO `products` (`id`, `sku`, `title`, `description`, `base_price`, `brand_id`, `weight_gram`, `is_published`) VALUES
('p0000001-0000-0000-0000-000000000001', 'SEM-BMS-005', 'Beras Merah Organik Tabanan 5 Kg', 'Beras merah kualitas premium asli Tabanan Bali, kaya serat dan indeks glikemik rendah.', 95000.00, 'b0000001-0000-0000-0000-000000000001', 5000, 1),
('p0000001-0000-0000-0000-000000000002', 'BEV-KNT-250', 'Biji Kopi Arabika Kintamani Honey Process 250g', 'Single origin Kintamani roast medium-to-dark dengan tasting notes citrus dan caramel.', 68000.00, 'b0000001-0000-0000-0000-000000000002', 250, 1),
('p0000001-0000-0000-0000-000000000003', 'ACC-LOG-M22', 'Logitech M220 Silent Wireless Mouse', 'Mouse nirkabel hening 2.4 GHz dengan baterai tahan hingga 18 bulan.', 179000.00, 'b0000001-0000-0000-0000-000000000003', 150, 1),
('p0000001-0000-0000-0000-000000000004', 'ACC-LOG-K38', 'Logitech K380 Multi-Device Bluetooth Keyboard', 'Keyboard bluetooth minimalis untuk Windows, Mac, Chrome OS, Android, dan iOS.', 459000.00, 'b0000001-0000-0000-0000-000000000003', 420, 1),
('p0000001-0000-0000-0000-000000000005', 'SEM-MNG-002', 'Minyak Goreng Sawit Murni 2 Liter', 'Minyak kelapa sawit higienis dua kali penyaringan kaya vitamin A & E.', 34000.00, 'b0000001-0000-0000-0000-000000000001', 2000, 1),
('p0000001-0000-0000-0000-000000000006', 'SEM-GLP-001', 'Gula Pasir Tebu Premium 1 Kg', 'Gula pasir kristal putih murni dari tebu pilihan Indonesia.', 17500.00, 'b0000001-0000-0000-0000-000000000001', 1000, 1),
('p0000001-0000-0000-0000-000000000007', 'BEV-KNT-100', 'Bubuk Kopi Kintamani Drip Bag (Isi 10 Sachet)', 'Kopi arabika celup praktis kualitas specialty siap seduh kapan saja.', 55000.00, 'b0000001-0000-0000-0000-000000000002', 120, 1),
('p0000001-0000-0000-0000-000000000008', 'ELC-SAM-A15', 'Samsung Galaxy A15 8/128GB LTE Blue', 'Smartphone layar 6.5 inci FHD+ Super AMOLED 90Hz, kamera 50MP triple.', 2499000.00, 'b0000001-0000-0000-0000-000000000006', 450, 1),
('p0000001-0000-0000-0000-000000000009', 'HOM-RIN-100', 'Rinso Molto Deterjen Cair Japanese Peach 1.8L', 'Deterjen konsentrat anti noda dengan keharuman tahan lama 21 hari.', 42000.00, 'b0000001-0000-0000-0000-000000000005', 1800, 1),
('p0000001-0000-0000-0000-000000000010', 'BAB-NUT-800', 'Nutrilon Royal 3 Vanila Tin 800g', 'Susu pertumbuhan formula Acti-Duobio+ untuk anak usia 1-3 tahun.', 215000.00, 'b0000001-0000-0000-0000-000000000004', 800, 1),
('p0000001-0000-0000-0000-000000000011', 'ACC-LOG-C92', 'Logitech C920 HD Pro Webcam', 'Full HD 1080p webcam dengan dual stereo microphone untuk video conference.', 1150000.00, 'b0000001-0000-0000-0000-000000000003', 300, 1),
('p0000001-0000-0000-0000-000000000012', 'HOM-SUN-750', 'Sunlight Jeruk Nipis 100 Pencuci Piring 750ml', 'Cairan pencuci piring ekstrak jeruk nipis asli membersihkan lemak 10x lebih cepat.', 16000.00, 'b0000001-0000-0000-0000-000000000005', 750, 1);
INSERT INTO `product_categories` (`product_id`, `category_id`) VALUES
('p0000001-0000-0000-0000-000000000001', 'c0000001-0000-0000-0000-000000000001'),
('p0000001-0000-0000-0000-000000000001', 'c0000001-0000-0000-0000-000000000007'),
('p0000001-0000-0000-0000-000000000002', 'c0000001-0000-0000-0000-000000000002'),
('p0000001-0000-0000-0000-000000000002', 'c0000001-0000-0000-0000-000000000008'),
('p0000001-0000-0000-0000-000000000003', 'c0000001-0000-0000-0000-000000000003'),
('p0000001-0000-0000-0000-000000000004', 'c0000001-0000-0000-0000-000000000003'),
('p0000001-0000-0000-0000-000000000005', 'c0000001-0000-0000-0000-000000000001'),
('p0000001-0000-0000-0000-000000000006', 'c0000001-0000-0000-0000-000000000001'),
('p0000001-0000-0000-0000-000000000007', 'c0000001-0000-0000-0000-000000000002'),
('p0000001-0000-0000-0000-000000000008', 'c0000001-0000-0000-0000-000000000006'),
('p0000001-0000-0000-0000-000000000009', 'c0000001-0000-0000-0000-000000000005'),
('p0000001-0000-0000-0000-000000000010', 'c0000001-0000-0000-0000-000000000004'),
('p0000001-0000-0000-0000-000000000011', 'c0000001-0000-0000-0000-000000000003'),
('p0000001-0000-0000-0000-000000000012', 'c0000001-0000-0000-0000-000000000005');
INSERT INTO `product_images` (`id`, `product_id`, `image_url`, `is_primary`, `display_order`) VALUES
('img00001-0000-0000-0000-000000000001', 'p0000001-0000-0000-0000-000000000001', 'https://images.unsplash.com/photo-1586201375761-83865001e31c?w=500', 1, 1),
('img00001-0000-0000-0000-000000000002', 'p0000001-0000-0000-0000-000000000002', 'https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=500', 1, 1),
('img00001-0000-0000-0000-000000000003', 'p0000001-0000-0000-0000-000000000003', 'https://images.unsplash.com/photo-1527864550417-7fd91fc51a46?w=500', 1, 1),
('img00001-0000-0000-0000-000000000004', 'p0000001-0000-0000-0000-000000000004', 'https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=500', 1, 1),
('img00001-0000-0000-0000-000000000005', 'p0000001-0000-0000-0000-000000000008', 'https://images.unsplash.com/photo-1598327105666-5b89351aff97?w=500', 1, 1);
INSERT INTO `product_variants` (`id`, `product_id`, `variant_sku`, `additional_price`) VALUES
('var00001-0000-0000-0000-000000000001', 'p0000001-0000-0000-0000-000000000003', 'ACC-LOG-M22-BLK', 0.00),
('var00001-0000-0000-0000-000000000002', 'p0000001-0000-0000-0000-000000000003', 'ACC-LOG-M22-RED', 5000.00),
('var00001-0000-0000-0000-000000000003', 'p0000001-0000-0000-0000-000000000004', 'ACC-LOG-K38-OFFW', 15000.00);
INSERT INTO `product_barcodes` (`id`, `product_id`, `barcode_number`, `barcode_type`) VALUES
('bar00001-0000-0000-0000-000000000001', 'p0000001-0000-0000-0000-000000000001', '8991234567011', 'EAN-13'),
('bar00001-0000-0000-0000-000000000002', 'p0000001-0000-0000-0000-000000000002', '8991234567028', 'EAN-13'),
('bar00001-0000-0000-0000-000000000003', 'p0000001-0000-0000-0000-000000000003', '097855123456', 'UPC-A'),
('bar00001-0000-0000-0000-000000000004', 'p0000001-0000-0000-0000-000000000004', '097855654321', 'UPC-A');
INSERT INTO `product_reviews` (`id`, `product_id`, `user_id`, `rating`, `review_text`) VALUES
('rev00001-0000-0000-0000-000000000001', 'p0000001-0000-0000-0000-000000000001', 'u0000001-0000-0000-0000-000000000003', 5, 'Beras merah wangi dan pulen sekali, cocok untuk program diet sehat keluarga.'),
('rev00001-0000-0000-0000-000000000002', 'p0000001-0000-0000-0000-000000000002', 'u0000001-0000-0000-0000-000000000004', 5, 'Aroma citrus kopi Kintamani terasa sangat khas saat diseduh V60 pagi hari.'),
('rev00001-0000-0000-0000-000000000003', 'p0000001-0000-0000-0000-000000000003', 'u0000001-0000-0000-0000-000000000003', 4, 'Klik mouse benar-benar senyap, nyaman digunakan di kafe.');

SET FOREIGN_KEY_CHECKS = 1;
