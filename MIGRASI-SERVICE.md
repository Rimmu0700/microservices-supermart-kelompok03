# Memecah Monolit Backend Menjadi Service Mandiri

Dosen meminta **semuanya dipisah**: kode, struktur, Dockerfile, database, dan pilihan teknologi per service
(misalnya service A memakai PostgreSQL dan service B memakai MySQL, masing-masing dengan struktur sendiri).
Dokumen ini menjelaskan apa yang berubah, cara menjalankan, dan apa yang MASIH harus dikerjakan.

## 1. Sebelum dan sesudah

| Aspek | Sebelum | Sesudah |
|---|---|---|
| Kode | satu folder `backend/`, satu `go.mod`, paket `pkg/` dan `internal/messaging` dipakai bersama | `services/<nama>/` masing-masing punya `go.mod`, `go.sum`, dan salinan kode yang ia butuhkan |
| Build | satu `backend/Dockerfile` + `ARG SERVICE` | `services/<nama>/Dockerfile` milik sendiri (multi-stage, non-root, HEALTHCHECK) |
| Database | 1 server MySQL berisi 9 database | 1 container database per service, volume + port + kredensial sendiri |
| Skema | `deployments/sql/generated/*.sql` (terpusat) | `services/<nama>/db/schema.sql` (milik service) |
| Konfigurasi | `x-common-env` bersama | `services/<nama>/.env.example` per service |
| Dokumentasi | README tunggal | `services/<nama>/README.md`: upstream, downstream, event, replika, kontrak |
| Kontrak gRPC | `backend/proto` | `proto/catalog/v1/catalog.proto` (sumber kontrak) + stub hasil kompilasi disalin ke catalog (provider) dan order (consumer) |
| Alat uji | `backend/cmd/grpcclient`, `payloadsize` | `tools/grpc-tools/` (modul terpisah) |

Pemisahan dilakukan lewat `scripts/split_services.py`. Skrip ini hanya **menyalin**; `backend/` asli tidak diubah,
jadi masih bisa dipakai sebagai cadangan.

## 2. Cara memakai

```bash
git checkout -b feat/service-split
# salin folder services/, tools/, scripts/split_services.py, docker-compose.microservices.yml ke root repo
docker compose down -v
docker compose -f docker-compose.microservices.yml up --build -d
docker compose -f docker-compose.microservices.yml ps     # service Go akan berstatus (healthy)
```
Bila kode di `backend/` berubah dan ingin menyalin ulang: `python3 scripts/split_services.py`.

Verifikasi bahwa tiap service benar-benar mandiri (bangun satu per satu, tanpa folder lain):
```bash
docker build -t order-service services/order-service
docker build -t catalog-service services/catalog-service
```

Skrip demo (`scripts-demo`) mengenali compose ini otomatis (nama container tetap `auth-service`, `nusantara-auth-db`, dst.).

## 3. Memisahkan lagi menjadi repositori Git sendiri (opsional)
```bash
git subtree split -P services/order-service -b split/order-service
git push git@github.com:<akun>/order-service.git split/order-service:main
```
Tiap folder sudah berisi semua yang dibutuhkan (kode, Dockerfile, skema DB, README), jadi bisa dipindah begitu saja.

## 4. Cara satu service pindah ke PostgreSQL (kebebasan teknologi)
Karena tiap service sekarang memegang salinan kodenya sendiri, pemilik service boleh mengubah database **tanpa menyentuh service lain**.
Daftar perubahan di dalam `services/<nama>/` (hasil pemeriksaan kode, bukan perkiraan):

1. **Driver dan koneksi**: `pkg/database/mysql.go` memakai `sqlx.Connect("mysql", ...)`. Tambahkan `ConnectPostgres`
   (`go get github.com/jackc/pgx/v5`, driver sqlx `pgx`, DSN `postgres://user:pass@host:5432/db?sslmode=disable`).
2. **Placeholder**: seluruh repository memakai `?`. PostgreSQL memakai `$1, $2, ...`. Pakai `db.Rebind(query)` dari sqlx
   (paling aman: bungkus akses DB agar selalu melewati `Rebind`).
3. **`ON DUPLICATE KEY UPDATE`** &rarr; `ON CONFLICT (kolom) DO UPDATE SET kol = EXCLUDED.kol`. Muncul di:
   `internal/auth/repository.go`, `internal/logistics/repository.go`, `internal/inventory/repository.go`,
   `internal/support/repository.go`, dan `pkg/replica/replica.go` (upsert replika; ada di SEMUA service pemakai replika).
4. **`INSERT IGNORE`** &rarr; `INSERT ... ON CONFLICT DO NOTHING`: `internal/logistics/consumer.go` dan `repository.go`.
5. **Deteksi duplikat**: `internal/messaging/ledger.go` memeriksa kode error MySQL 1062; di PostgreSQL ganti dengan SQLSTATE `23505`.
6. **Fungsi tanggal**: `DATE_*` di `internal/logistics/repository.go` &rarr; aritmetika `INTERVAL`. `NOW()` dan `LIMIT` sudah kompatibel.
7. **Skema**: tulis ulang `db/schema.sql`: hapus `ENGINE=InnoDB` dan backtick, `DATETIME` &rarr; `TIMESTAMP`, `JSON` &rarr; `JSONB`,
   `TINYINT(1)` &rarr; `BOOLEAN`.
8. **Compose**: container DB pakai `postgres:16-alpine`, env `POSTGRES_DB/USER/PASSWORD`, port 5432,
   healthcheck `pg_isready -U <user> -d <db>`, skema dipasang ke `/docker-entrypoint-initdb.d/`.

**Kandidat termudah: `promotion-service`.** Dari hasil `grep`, repository-nya hanya memakai `NOW()` dan `LIMIT`
(tidak ada `ON DUPLICATE`); yang perlu diubah hanya placeholder, `pkg/replica`, dan skema. Periksa dulu apakah ia memakai ledger idempotensi.

> Konversi ini **belum saya kerjakan** karena butuh kompilasi dan pengujian Go yang tidak tersedia di lingkungan saya.
> Struktur sekarang sudah memungkinkannya secara terisolasi. Jika dikerjakan, uji dengan `docker compose ... up --build` dan skrip demo.

## 5. Batasan yang jujur
- **Duplikasi kode**: `internal/messaging`, `pkg/bootstrap`, dst. kini disalin ke tiap service. Ini harga independensi
  (tidak ada kopling build). Jalur lanjutan: jadikan paket bersama pustaka berversi di repositori terpisah.
- **Rahasia bersama**: `JWT_SECRET` dan `INTERNAL_TOKEN` masih sama untuk semua service, dan satu akun RabbitMQ.
  Perbaikan: JWT bertanda tangan kunci privat (hanya auth) dengan kunci publik di service lain; token internal dan akun broker per service.
- **Nama modul Go** masih `github.com/nusantara-supermart/backend` di tiap `go.mod`. Aman karena modul terpisah;
  ganti per service bila dipecah menjadi repositori sendiri.
- **Belum dikompilasi di lingkungan saya** (tidak ada Go/Docker/jaringan). Pemeriksaan yang dilakukan: setiap impor internal
  terpenuhi di dalam folder service, tidak ada impor lintas service, struktur compose valid. Verifikasi build penuh ada di langkah 2.
- Frontend dan gateway tetap di `frontend/` dan `deployments/gateway/` (sudah mandiri).

## 6. Tambahan Tugas 5 (API Gateway)
`docker-compose.microservices.yml` kini memakai **KrakenD** (`api-gateway`, port 8000) menggantikan nginx `gateway`, dan menghapus port host
service, database, serta AMQP (konsol RabbitMQ hanya `127.0.0.1:15672`). Konfigurasi gateway ada di `gateway/` dan dijelaskan di
`docs/tugas5-api-gateway.md`. Port gRPC catalog tertutup; pakai `docker-compose.grpc-debug.yml` untuk klien/benchmark gRPC dari host.
Pemecah service (`scripts/split_services.py`) sudah menghasilkan compose versi ini.
