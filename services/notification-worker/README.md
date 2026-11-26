# notification-worker

**Peran:** Consumer murni: mengirim notifikasi WhatsApp/Email (simulasi log) dari event order dan inventory.

| Item | Nilai |
|---|---|
| Pemilik (anggota tim) | rizkyandika |
| Modul Go | `github.com/nusantara-supermart/backend` (go.mod milik folder ini) |
| Port HTTP | 3000 (di dalam jaringan Docker) |
| Database milik service ini | `notification_db` pada container `nusantara-notification-db` (MySQL 8.0, port host 3320) |
| Skema database | `db/schema.sql` (MILIK service ini; boleh diganti dialeknya tanpa memengaruhi service lain) |
| Dockerfile | `Dockerfile` (multi-stage, non-root UID 10001) |

## Upstream (pihak yang menyediakan data untuk service ini)
- order (event order.#)
- inventory (event inventory.#)

## Downstream (pihak yang bergantung pada service ini)
- (tidak ada)

## Kontrak komunikasi
**Event yang diterbitkan:**
- (tidak ada)

**Event yang dikonsumsi:**
- order.# dan inventory.# (notification.order-events.q)

**Antarmuka yang disediakan:**
- (tidak ada)

**Memanggil service lain:**
- (tidak ada)

**Replika data referensi (pull berkala dari pemilik data):**
- (tidak ada)

**Data yang diekspor ke service lain (GET /internal/replica/:table):**
- (tidak ada)

## Menjalankan sendiri
```bash
docker build -t notification-worker .                       # build image milik service ini saja
go build ./cmd/notification-worker                          # atau build lokal
```
Menjalankan seluruh sistem: dari root repositori `docker compose -f docker-compose.microservices.yml up --build -d`.

## Mengganti teknologi database (kebebasan service ini)
Semua yang perlu diubah ada **di dalam folder ini saja**: `pkg/database/` (driver dan DSN), `internal/notification/`
dan `internal/messaging/` + `pkg/replica/` (dialek SQL), `db/schema.sql`, serta definisi container DB-nya di compose.
Lihat `MIGRASI-SERVICE.md` di root untuk daftar perubahan ke PostgreSQL.
