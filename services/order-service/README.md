# order-service

**Peran:** Keranjang, checkout, status order. Menerbitkan OrderPlaced lewat Transactional Outbox.

| Item | Nilai |
|---|---|
| Pemilik (anggota tim) | ______________ |
| Modul Go | `github.com/nusantara-supermart/backend` (go.mod milik folder ini) |
| Port HTTP | 3000 (di dalam jaringan Docker) |
| Database milik service ini | `order_db` pada container `nusantara-order-db` (MySQL 8.0, port host 3314) |
| Skema database | `db/schema.sql` (MILIK service ini; boleh diganti dialeknya tanpa memengaruhi service lain) |
| Dockerfile | `Dockerfile` (multi-stage, non-root UID 10001) |

## Upstream (pihak yang menyediakan data untuk service ini)
- auth
- catalog (gRPC + replika)
- inventory (replika warehouses, event gagal reservasi)

## Downstream (pihak yang bergantung pada service ini)
- payment
- logistics
- inventory
- support
- notification-worker

## Kontrak komunikasi
**Event yang diterbitkan:**
- OrderPlaced -> order.<wilayah>.placed

**Event yang dikonsumsi:**
- StockReservationFailed (order.reservation-failed.q)

**Antarmuka yang disediakan:**
- REST publik /api/v1/orders/*
- REST internal: PUT /internal/orders/:id/status (dipanggil payment dan logistics)

**Memanggil service lain:**
- gRPC -> catalog-service:50051 (GetProduct)
- REST internal -> inventory-service (release stok)

**Replika data referensi (pull berkala dari pemilik data):**
- users, user_addresses dari auth
- products, product_images dari catalog
- warehouses dari inventory

**Data yang diekspor ke service lain (GET /internal/replica/:table):**
- orders
- order_shipping_details

## Menjalankan sendiri
```bash
docker build -t order-service .                       # build image milik service ini saja
go build ./cmd/order-service                          # atau build lokal
```
Menjalankan seluruh sistem: dari root repositori `docker compose -f docker-compose.microservices.yml up --build -d`.

## Mengganti teknologi database (kebebasan service ini)
Semua yang perlu diubah ada **di dalam folder ini saja**: `pkg/database/` (driver dan DSN), `internal/order/`
dan `internal/messaging/` + `pkg/replica/` (dialek SQL), `db/schema.sql`, serta definisi container DB-nya di compose.
Lihat `MIGRASI-SERVICE.md` di root untuk daftar perubahan ke PostgreSQL.
