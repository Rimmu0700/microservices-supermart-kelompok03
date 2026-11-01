-- =============================================================================
-- Tugas 3: tabel pendukung Event-Driven Architecture
-- =============================================================================

-- 1) Idempotency Ledger (Processed Events Table)
--    Kunci (consumer_name, event_id): satu database dipakai bersama oleh beberapa consumer,
--    sehingga setiap consumer harus punya "buku catat" sendiri.
CREATE TABLE IF NOT EXISTS `processed_events` (
    `consumer_name` VARCHAR(100) NOT NULL,
    `event_id`      VARCHAR(36)  NOT NULL,
    `event_type`    VARCHAR(100) NOT NULL,
    `processed_at`  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`consumer_name`, `event_id`)
) ENGINE=InnoDB;

-- 2) Transactional Outbox: event ditulis di transaksi yang sama dengan data bisnis
CREATE TABLE IF NOT EXISTS `outbox_events` (
    `id`             VARCHAR(36)  PRIMARY KEY,           -- sama dengan event_id
    `aggregate_type` VARCHAR(50)  NOT NULL,
    `aggregate_id`   VARCHAR(36)  NOT NULL,
    `event_type`     VARCHAR(100) NOT NULL,
    `routing_key`    VARCHAR(255) NOT NULL,
    `payload`        JSON         NOT NULL,
    `created_at`     DATETIME(3)  DEFAULT CURRENT_TIMESTAMP(3),
    `published_at`   DATETIME(3)  NULL,
    INDEX `idx_outbox_unpublished` (`published_at`, `created_at`)
) ENGINE=InnoDB;