# logistics-service

**Peran:** Pengiriman, kurir, bukti terima (POD).

| Item | Nilai |
|---|---|
| Pemilik (anggota tim) | ______________ |
| Modul Go | `github.com/nusantara-supermart/backend` (go.mod milik folder ini) |
| Port HTTP | 3000 (di dalam jaringan Docker) |
| Database milik service ini | `logistics_db` pada container `nusantara-logistics-db` (MySQL 8.0, port host 3317) |
| Skema database | `db/schema.sql` (MILIK service ini; boleh diganti dialeknya tanpa memengaruhi service lain) |
| Dockerfile | `Dockerfile` (multi-stage, non-root UID 10001) |

## Upstream (pihak yang menyediakan data untuk service ini)
- order (event OrderPlaced, replika orders dan order_shipping_details)
- auth (replika users, alamat)

## Downstream (pihak yang bergantung pada service ini)
- (tidak ada)

## Kontrak komunikasi
**Event yang diterbitkan:**
- (tidak ada)

**Event yang dikonsumsi:**
- OrderPlaced (logistics.shipment-creation.q)

**Antarmuka yang disediakan:**
- REST publik /api/v1/logistics/*

**Memanggil service lain:**
- REST internal -> order-service (SHIPPED / COMPLETED)

**Replika data referensi (pull berkala dari pemilik data):**
- orders, order_shipping_details dari order
- users, user_addresses dari auth

**Data yang diekspor ke service lain (GET /internal/replica/:table):**
- (tidak ada)

## Menjalankan sendiri
```bash
docker build -t logistics-service .                       # build image milik service ini saja
go build ./cmd/logistics-service                          # atau build lokal
```
Menjalankan seluruh sistem: dari root repositori `docker compose -f docker-compose.yml up --build -d`.

## Mengganti teknologi database (kebebasan service ini)
Semua yang perlu diubah ada **di dalam folder ini saja**: `pkg/database/` (driver dan DSN), `internal/logistics/`
dan `internal/messaging/` + `pkg/replica/` (dialek SQL), `db/schema.sql`, serta definisi container DB-nya di compose.
Lihat `MIGRASI-SERVICE.md` di root untuk daftar perubahan ke PostgreSQL.
