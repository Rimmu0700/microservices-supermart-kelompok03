# procurement-service

**Peran:** Purchase order dan penerimaan barang (GRN) dari pemasok. Menerbitkan GoodsReceived lewat Outbox.

| Item | Nilai |
|---|---|
| Pemilik (anggota tim) | rizkyandika |
| Modul Go | `github.com/nusantara-supermart/backend` (go.mod milik folder ini) |
| Port HTTP | 3000 (di dalam jaringan Docker) |
| Database milik service ini | `procurement_db` pada container `nusantara-procurement-db` (MySQL 8.0, port host 3318) |
| Skema database | `db/schema.sql` (MILIK service ini; boleh diganti dialeknya tanpa memengaruhi service lain) |
| Dockerfile | `Dockerfile` (multi-stage, non-root UID 10001) |

## Upstream (pihak yang menyediakan data untuk service ini)
- catalog (replika products)
- inventory (replika warehouses)

## Downstream (pihak yang bergantung pada service ini)
- inventory (event GoodsReceived)

## Kontrak komunikasi
**Event yang diterbitkan:**
- GoodsReceived -> procurement.<wilayah>.goods_received

**Event yang dikonsumsi:**
- (tidak ada)

**Antarmuka yang disediakan:**
- REST publik /api/v1/procurement/*

**Memanggil service lain:**
- (tidak ada)

**Replika data referensi (pull berkala dari pemilik data):**
- products dari catalog
- warehouses dari inventory

**Data yang diekspor ke service lain (GET /internal/replica/:table):**
- (tidak ada)

## Menjalankan sendiri
```bash
docker build -t procurement-service .                       # build image milik service ini saja
go build ./cmd/procurement-service                          # atau build lokal
```
Menjalankan seluruh sistem: dari root repositori `docker compose -f docker-compose.yml up --build -d`.

## Mengganti teknologi database (kebebasan service ini)
Semua yang perlu diubah ada **di dalam folder ini saja**: `pkg/database/` (driver dan DSN), `internal/procurement/`
dan `internal/messaging/` + `pkg/replica/` (dialek SQL), `db/schema.sql`, serta definisi container DB-nya di compose.
Lihat `MIGRASI-SERVICE.md` di root untuk daftar perubahan ke PostgreSQL.
