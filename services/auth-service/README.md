# auth-service

**Peran:** Registrasi, login, penerbitan JWT, alamat pengguna, role dan permission.

| Item | Nilai |
|---|---|
| Pemilik (anggota tim) | Maria Dwi Rosari Nahak |
| Modul Go | `github.com/nusantara-supermart/backend` (go.mod milik folder ini) |
| Port HTTP | 3000 (di dalam jaringan Docker) |
| Database milik service ini | `auth_db` pada container `nusantara-auth-db` (MySQL 8.0, port host 3311) |
| Skema database | `db/schema.sql` (MILIK service ini; boleh diganti dialeknya tanpa memengaruhi service lain) |
| Dockerfile | `Dockerfile` (multi-stage, non-root UID 10001) |

## Upstream (pihak yang menyediakan data untuk service ini)
- (tidak ada)

## Downstream (pihak yang bergantung pada service ini)
- catalog
- inventory
- order
- payment
- logistics
- support

## Kontrak komunikasi
**Event yang diterbitkan:**
- (tidak ada)

**Event yang dikonsumsi:**
- (tidak ada)

**Antarmuka yang disediakan:**
- REST publik /api/v1/auth/*

**Memanggil service lain:**
- (tidak ada)

**Replika data referensi (pull berkala dari pemilik data):**
- (tidak ada)

**Data yang diekspor ke service lain (GET /internal/replica/:table):**
- users (tanpa password_hash)
- user_addresses
- roles
- user_roles

## Menjalankan sendiri
```bash
docker build -t auth-service .                       # build image milik service ini saja
go build ./cmd/auth-service                          # atau build lokal
```
Menjalankan seluruh sistem: dari root repositori `docker compose -f docker-compose.microservices.yml up --build -d`.

## Mengganti teknologi database (kebebasan service ini)
Semua yang perlu diubah ada **di dalam folder ini saja**: `pkg/database/` (driver dan DSN), `internal/auth/`
dan `internal/messaging/` + `pkg/replica/` (dialek SQL), `db/schema.sql`, serta definisi container DB-nya di compose.
Lihat `MIGRASI-SERVICE.md` di root untuk daftar perubahan ke PostgreSQL.
