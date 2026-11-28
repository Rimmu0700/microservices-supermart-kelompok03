# promotion-service

**Peran:** Voucher dan promo.

| Item | Nilai |
|---|---|
| Pemilik (anggota tim) | ______________ |
| Modul Go | `github.com/nusantara-supermart/backend` (go.mod milik folder ini) |
| Port HTTP | 3000 (di dalam jaringan Docker) |
| Database milik service ini | `promotion_db` pada container `nusantara-promotion-db` (MySQL 8.0, port host 3316) |
| Skema database | `db/schema.sql` (MILIK service ini; boleh diganti dialeknya tanpa memengaruhi service lain) |
| Dockerfile | `Dockerfile` (multi-stage, non-root UID 10001) |

## Upstream (pihak yang menyediakan data untuk service ini)
- catalog (replika products, product_images)

## Downstream (pihak yang bergantung pada service ini)
- (tidak ada)

## Kontrak komunikasi
**Event yang diterbitkan:**
- (tidak ada)

**Event yang dikonsumsi:**
- (tidak ada)

**Antarmuka yang disediakan:**
- REST publik /api/v1/promotions/*

**Memanggil service lain:**
- (tidak ada)

**Replika data referensi (pull berkala dari pemilik data):**
- products, product_images dari catalog

**Data yang diekspor ke service lain (GET /internal/replica/:table):**
- (tidak ada)

## Menjalankan sendiri
```bash
docker build -t promotion-service .                       # build image milik service ini saja
go build ./cmd/promotion-service                          # atau build lokal
```
Menjalankan seluruh sistem: dari root repositori `docker compose -f docker-compose.yml up --build -d`.

## Mengganti teknologi database (kebebasan service ini)
Semua yang perlu diubah ada **di dalam folder ini saja**: `pkg/database/` (driver dan DSN), `internal/promotion/`
dan `internal/messaging/` + `pkg/replica/` (dialek SQL), `db/schema.sql`, serta definisi container DB-nya di compose.
Lihat `MIGRASI-SERVICE.md` di root untuk daftar perubahan ke PostgreSQL.
