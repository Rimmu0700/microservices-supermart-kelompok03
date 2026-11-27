# Skrip Demo UTS - Arsitektur Mikroservis (Kelompok 03)

Satu skrip = satu tujuan, gaya yang sama dengan `scripts/smoke-test.sh`. Setiap skrip berdiri sendiri
(tidak saling memanggil) dan mencetak `[PASS]` / `[FAIL]` serta nama host + waktu untuk screenshot.

## Cara pasang
Salin folder ini ke repositori sebagai `scripts/demo/`, lalu jalankan dari mana saja di dalam repo:

    bash scripts/demo/00-cek-prasyarat-sebelum-demo.sh

Windows: jalankan di **Git Bash** atau **WSL** (bukan PowerShell biasa). Perlu: docker, curl, python3.
Go (skrip 02) dan k6 (skrip 02 cadangan dan 03) hanya untuk bagian gRPC.

## Daftar skrip

| Berkas | Jenis | Bagian | Fungsi | Lama |
|---|---|---|---|---|
| `00-cek-prasyarat-sebelum-demo.sh` | Persiapan | - | Cek alat, 14 container, gateway, login, RabbitMQ, MySQL, port gRPC | 10 dtk |
| `01-A-bukti-alur-transaksi-end-to-end.sh` | PEMBUKTIAN | A | Satu pembelian utuh: gRPC (add-to-cart) + event (faktur, pengiriman) + bayar, plus jejak outbox/ledger | 30 dtk |
| `02-B1-bukti-grpc-server-klien-dan-ukuran-payload.sh` | PEMBUKTIAN | B1-B3 | Server gRPC hidup, klien Unary+Batch+NotFound, ukuran JSON vs Protobuf | 30 dtk |
| `03-B2-benchmark-rest-vs-grpc-k6.sh` | PEMBUKTIAN | B4 | k6 REST vs gRPC (50 VU/30s dan 200 VU/60s) + tabel perbandingan; hasil di `hasil-demo/` | ~3 mnt |
| `04-C-bukti-rabbitmq-topologi-dan-event.sh` | PEMBUKTIAN | C | Exchange, queue, consumer, binding; 1 checkout; outbox terbit; ledger idempotensi; queue kosong | 1 mnt |
| `10-D1-gangguan-consumer-asinkron-mati.sh` | GANGGUAN | D1 | Matikan inventory-service, 50 checkout, Ready naik, payment tetap jalan, pulih, zero loss | ~2 mnt |
| `11-D2-gangguan-provider-grpc-catalog-mati.sh` | GANGGUAN | D2 | Matikan catalog-service: add-to-cart gagal tertata, service lain hidup, pulih | 1 mnt |
| `12-D3-gangguan-broker-rabbitmq-mati.sh` | GANGGUAN | D3 | Matikan RabbitMQ: checkout tetap sukses, event tertahan di outbox, terbit setelah pulih | 2 mnt |
| `13-D4-gangguan-poison-pill-ke-dlq.sh` | GANGGUAN | D4 | Suntik pesan rusak: ditolak, masuk `supermart.dlq` | 30 dtk |
| `14-D5-gangguan-container-acak-pilihan-dosen.sh` | GANGGUAN | D5 | Matikan 1 container (acak atau pilihan): tabel sebelum / saat mati / sesudah + prediksi dampak | 1-2 mnt |
| `20-E-bukti-isolasi-data-database-per-service.sh` | PEMBUKTIAN | E | 10 database, user per database, nol FK lintas DB, snapshot harga, tabel outbox/ledger | 10 dtk |

Penamaan: `<nomor>-<bagian>-<jenis>-<apa yang dilakukan>.sh`. Nomor 0x = pembuktian, 1x = simulasi gangguan, 2x = tambahan.

## Urutan latihan yang disarankan
`00` -> `01` -> `02` -> `04` -> `20` -> `10` -> `11` -> `12` -> `13` -> `14` (acak) -> `03` (terakhir, paling lama).

## Opsi (environment variable)
- `COUNT=10 bash scripts/demo/10-D1-...sh` : jumlah checkout (bawaan 50, sesuai soal Tugas 3)
- `QUICK=1 bash scripts/demo/03-B2-...sh` : benchmark cepat 10 VU / 5 detik
- `bash scripts/demo/14-D5-...sh payment-service` : pilih container; `--list` untuk daftar
- `AUTO_RESTART=1` (skrip 12), `CLEAN=1` (skrip 13)

## Pengaman
Skrip gangguan (10-14) otomatis menyalakan kembali container yang dimatikan bila dihentikan dengan Ctrl+C.
Jika tetap ada yang mati: `docker compose up -d`.

## Catatan kejujuran
Skrip ini sudah dicek sintaksisnya dan diuji alurnya dengan Docker/curl tiruan, tetapi **belum dijalankan terhadap
sistem asli kalian**. Jalankan semuanya sekali sebelum hari H. Pemeriksaan bertanda `[PERHATIAN]` bukan kegagalan
skrip, melainkan hal yang perlu kalian jelaskan atau selidiki. Nama queue, container, kredensial, dan akun uji
diambil dari `docker-compose.yml` dan `scripts/smoke-test.sh`; ubah lewat environment variable bila berbeda.

## Dua mode database (terdeteksi otomatis)
- **shared**: `docker-compose.yml` bawaan, 1 container `nusantara-mysql` berisi 9 database.
- **per-service**: `docker-compose.db-per-service.yml`, 1 container MySQL per service (`nusantara-auth-db`, ...).
Skrip 00, 14, dan 20 menyesuaikan diri; skrip lain mengarahkan kueri database ke container yang benar.
Pindah mode: `docker compose down`, lalu `docker compose -f docker-compose.db-per-service.yml up --build -d`.

## Compose yang dikenali
`docker-compose.yml` (bawaan lama, gateway nginx :3000) dan `docker-compose.microservices.yml` (semua service mandiri + API Gateway KrakenD :8000).
Skrip membaca compose yang sedang dipakai dari label container, jadi tidak perlu `-f` atau `COMPOSE_FILE` manual.

## Perubahan Tugas 5 (API Gateway)
- Alamat API bawaan skrip kini `http://localhost:8000/api/v1` (KrakenD). Ganti dengan `BASE=...` bila perlu.
- Port gRPC catalog tidak lagi dibuka ke host. Skrip 02 dan 03 memberi petunjuk memakai `docker-compose.grpc-debug.yml` (override sementara).
- Skrip khusus Tugas 5 ada di `scripts/gateway/` (routing, isolasi, agregasi, rate limit, latensi).
