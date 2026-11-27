# catalog-service

**Peran:** Produk, kategori, brand, gambar, ulasan. Menyediakan REST dan gRPC (GetProduct, BatchGetProducts).

| Item | Nilai |
|---|---|
| Pemilik (anggota tim) | ______________ |
| Modul Go | `github.com/nusantara-supermart/backend` (go.mod milik folder ini) |
| Port HTTP | 3000 (di dalam jaringan Docker) |
| Port gRPC | 50051 |
| Database milik service ini | `catalog_db` pada container `nusantara-catalog-db` (MySQL 8.0, port host 3312) |
| Skema database | `db/schema.sql` (MILIK service ini; boleh diganti dialeknya tanpa memengaruhi service lain) |
| Dockerfile | `Dockerfile` (multi-stage, non-root UID 10001) |

## Upstream (pihak yang menyediakan data untuk service ini)
- auth (replika users)

## Downstream (pihak yang bergantung pada service ini)
- order (gRPC + replika)
- inventory
- promotion
- procurement

## Kontrak komunikasi
**Event yang diterbitkan:**
- (tidak ada)

**Event yang dikonsumsi:**
- (tidak ada)

**Antarmuka yang disediakan:**
- REST publik /api/v1/catalog/*
- gRPC :50051 supermart.catalog.v1.CatalogService

**Memanggil service lain:**
- (tidak ada)

**Replika data referensi (pull berkala dari pemilik data):**
- users dari auth

**Data yang diekspor ke service lain (GET /internal/replica/:table):**
- products
- product_images

## Menjalankan sendiri
```bash
docker build -t catalog-service .                       # build image milik service ini saja
go build ./cmd/catalog-service                          # atau build lokal
```
Menjalankan seluruh sistem: dari root repositori `docker compose -f docker-compose.microservices.yml up --build -d`.

## Mengganti teknologi database (kebebasan service ini)
Semua yang perlu diubah ada **di dalam folder ini saja**: `pkg/database/` (driver dan DSN), `internal/catalog/`
dan `internal/messaging/` + `pkg/replica/` (dialek SQL), `db/schema.sql`, serta definisi container DB-nya di compose.
Lihat `MIGRASI-SERVICE.md` di root untuk daftar perubahan ke PostgreSQL.
