-- =============================================================================
-- IDENTITY SERVICE — DATA AWAL (SEED)
-- Master data RBAC (5 peran, 9 izin) + 1 akun Super Admin untuk pengujian.
-- Password akun demo: password123 (bcrypt cost 10)
-- =============================================================================

USE `identity_service_db`;

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
('u0000001-0000-0000-0000-000000000001', 'admin@nusantara.co.id', '$2a$10$UwQm.bwc/69L3mHOIx89VOgaeQhi9cJQvtTm.Y5/AbW2P7NRqgfBC', 'Budi Hartono (Admin)', '081122334455', 1);

INSERT INTO `user_roles` (`user_id`, `role_id`) VALUES
('u0000001-0000-0000-0000-000000000001', 'r0000001-0000-0000-0000-000000000001');
