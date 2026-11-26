# inventory-service

**Peran:** Stok gudang dan reservasi stok.

| Item | Nilai |
|---|---|
| Pemilik (anggota tim) | rizkyandika |
| Modul Go | `github.com/nusantara-supermart/backend` (go.mod milik folder ini) |
| Port HTTP | 3000 (di dalam jaringan Docker) |
| Database milik service ini | `inventory_db` pada container `nusantara-inventory-db` (MySQL 8.0, port host 3313) |
| Skema database | `db/schema.sql` (MILIK service ini; boleh diganti dialeknya tanpa memengaruhi service lain) |
| Dockerfile | `Dockerfile` (multi-stage, non-root UID 10001) |

## Upstream (pihak yang menyediakan data untuk service ini)
- auth
- catalog (replika)
- order (event OrderPlaced)
- procurement (event GoodsReceived)

## Downstream (pihak yang bergantung pada service ini)
- order (replika warehouses, event gagal reservasi, API release)
- procurement (replika warehouses)
- notification-worker

## Kontrak komunikasi
**Event yang diterbitkan:**
- StockReservationFailed -> inventory.<wilayah>.reservation_failed

**Event yang dikonsumsi:**
- OrderPlaced (inventory.stock-reservation.q)
- GoodsReceived (inventory.goods-received.q)

**Antarmuka yang disediakan:**
- REST publik /api/v1/inventory/*
- REST internal: release stok (dipanggil order)

**Memanggil service lain:**
- (tidak ada)

**Replika data referensi (pull berkala dari pemilik data):**
- users dari auth
- products dari catalog

**Data yang diekspor ke service lain (GET /internal/replica/:table):**
- warehouses

## Menjalankan sendiri
```bash
docker build -t inventory-service .                       # build image milik service ini saja
go build ./cmd/inventory-service                          # atau build lokal
```
Menjalankan seluruh sistem: dari root repositori `docker compose -f docker-compose.microservices.yml up --build -d`.

## Mengganti teknologi database (kebebasan service ini)
Semua yang perlu diubah ada **di dalam folder ini saja**: `pkg/database/` (driver dan DSN), `internal/inventory/`
dan `internal/messaging/` + `pkg/replica/` (dialek SQL), `db/schema.sql`, serta definisi container DB-nya di compose.
Lihat `MIGRASI-SERVICE.md` di root untuk daftar perubahan ke PostgreSQL.
