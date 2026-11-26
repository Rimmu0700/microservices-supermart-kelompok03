# Keputusan Arsitektur (Architecture Decision Records)

Proyek: Nusantara SuperMart - Kelompok 03 (Arsitektur Mikroservis, TI-3142).
Format tiap keputusan: Konteks, Keputusan, Alasan, Konsekuensi (termasuk kelemahan yang kami sadari).

## ADR-001 Dekomposisi menjadi 9 service + 1 worker, database-per-service
- **Konteks:** seed project berupa modular monolith dengan satu database dan JOIN lintas domain.
- **Keputusan:** memecah menjadi auth, catalog, inventory, order, payment, promotion, logistics, procurement, support, dan
  notification-worker. Tiap service memiliki kode, modul Go, Dockerfile, skema (`db/schema.sql`), dan container database sendiri
  (`docker-compose.microservices.yml`). Tidak ada foreign key atau JOIN lintas database; hubungan antar data hanya berupa ID.
- **Alasan:** loose coupling, deploy dan perubahan skema mandiri, isolasi kegagalan, hak akses minimal (user DB per service),
  dan kebebasan memilih teknologi data per service.
- **Konsekuensi:** data lintas domain harus diambil lewat gRPC, event, atau replika (ADR-002 sampai 005); konsistensi menjadi
  eventual; ada duplikasi kode pustaka bersama (`pkg/`, `internal/messaging`) di tiap service.

## ADR-002 Komunikasi sinkron memakai gRPC hanya pada jalur Order ke Catalog
- **Konteks:** saat add-to-cart, Order harus tahu produk valid dan harganya saat itu juga; jalur ini baca-berat.
- **Keputusan:** catalog-service menyediakan gRPC :50051 (`proto/catalog/v1/catalog.proto`: GetProduct unary dan
  BatchGetProducts batch). Harga memakai message `Money` (currency_code + amount int64), tanpa float.
- **Alasan:** HTTP/2 multiplexing dan koneksi persisten, Protobuf biner lebih ringkas, kontrak bertipe statis,
  batch mencegah chatty I/O.
- **Konsekuensi:** Order bergantung langsung pada ketersediaan Catalog (add-to-cart gagal bila Catalog mati; timeout 2 detik).
  Biner sulit diinspeksi; L7 load balancing lebih rumit; browser butuh gRPC-Web, maka gRPC hanya untuk East-West.
  `BatchGetProducts` sudah tersedia tetapi belum dipakai alur checkout.

## ADR-003 Komunikasi asinkron memakai RabbitMQ (topic exchange)
- **Konteks:** checkout menyentuh inventory, payment, logistics, dan notifikasi; bila sinkron, satu service lambat menjatuhkan checkout.
- **Keputusan:** exchange `supermart.events` (topic), routing key `<domain>.<wilayah>.<status>`, queue per consumer
  (mis. `inventory.stock-reservation.q` dengan binding `order.*.placed`). Queue dan exchange durable, pesan persistent,
  manual ACK, prefetch 10, serta Dead-Letter Exchange `supermart.deadletter.exchange` dengan antrean karantina `supermart.dlq`.
- **Alasan:** temporal decoupling, peredam lonjakan, satu event untuk banyak consumer, isolasi kegagalan.
- **Konsekuensi:** pengiriman at-least-once sehingga consumer wajib idempoten; hasil akhir baru konsisten beberapa detik kemudian.

## ADR-004 Transactional Outbox dan Idempotent Consumer
- **Konteks:** dual-write problem: INSERT DB berhasil tetapi publish ke broker gagal.
- **Keputusan:** Order dan Procurement menulis baris `outbox_events` pada transaksi bisnis yang sama; relay menerbitkan ke
  RabbitMQ dan mengisi `published_at`. Tiap consumer mencatat `(consumer_name, event_id)` pada `processed_events` dalam
  transaksi yang sama dengan mutasi bisnis. Kompensasi stok memakai Saga choreography (`StockReservationFailed`).
- **Alasan:** atomisitas tanpa Two-Phase Commit; duplikasi pesan tidak mengubah hasil bisnis.
- **Konsekuensi:** event bisa terbit lebih dari sekali (ditangani idempotensi); ada penundaan kecil dari polling relay.

## ADR-005 Replikasi data referensi (pull berkala)
- **Konteks:** tanpa JOIN lintas database, sebagian service tetap butuh data referensi (produk, pengguna, gudang, order).
- **Keputusan:** pemilik data membuka `GET /internal/replica/:table` (dilindungi `X-Internal-Token`, kolom sensitif seperti
  `password_hash` dikecualikan); pemakai menarik tiap ~3 detik dan upsert ke tabel lokal.
- **Konsekuensi:** data replika dapat tertinggal sekitar 3 detik (eventual consistency); skema tabel pemilik menjadi
  kontrak yang diikuti pemakai (hubungan Customer-Supplier/Conformist).

## ADR-006 Kontainer: multi-stage, non-root, minimal
- **Keputusan:** Dockerfile tiap service memakai builder `golang:1.26-alpine` dengan `CGO_ENABLED=0` dan `-ldflags="-s -w"`,
  runtime `alpine:3.19` dengan pengguna non-root (UID 10001) dan `HEALTHCHECK` ke `/api/v1/health`.
  Frontend memakai multi-stage ke nginx.
- **Alasan:** image kecil, permukaan serangan kecil, least privilege, status "healthy" terpantau.
- **Konsekuensi:** alpine tetap memiliki shell; varian distroless/scratch tersedia di `deployments/docker/` untuk perbandingan Tugas 4.

## ADR-007 Pilihan DBMS
- **Keputusan saat ini:** semua service memakai MySQL 8.0 (container terpisah per service) karena kode data-access ditulis dengan
  sqlx dan dialek MySQL sejak seed project.
- **Kebebasan teknologi:** karena kode `pkg/database`, `pkg/replica`, dan `internal/messaging` kini disalin per service,
  pemilik service dapat berpindah (mis. ke PostgreSQL) tanpa memengaruhi service lain. Daftar perubahan ada di `MIGRASI-SERVICE.md`.
- **Konsekuensi:** konversi belum dikerjakan; perlu ubah driver, placeholder, `ON DUPLICATE KEY`, deteksi duplikat, dan skema.

## ADR-008 API Gateway KrakenD sebagai pintu masuk tunggal (Tugas 5)
- **Konteks:** klien (frontend, Postman) sebelumnya menjangkau beberapa port (gateway nginx :3000, gRPC :50051, database, broker) dan
  harus mengetahui topologi internal; kebijakan lintas-layanan (CORS, pembatasan laju) tersebar.
- **Keputusan:** KrakenD (`nusantara-api-gateway`, port host 8000) menjadi satu-satunya pintu API untuk klien eksternal. Seluruh 101 rute
  REST dilewatkan pass-through (`no-op`), ditambah 2 alias path rewriting, 3 endpoint agregasi paralel (`storefront`, `order-summary`, `health`),
  dan rate limit token bucket per IP pada login (10/menit), register (5/menit), dan checkout (10/menit). Service dan database hanya `expose`
  di jaringan Docker; frontend meneruskan `/api/` ke gateway. Konfigurasi dipecah per domain (`gateway/routes/<domain>.json`) dan digabung
  oleh `gateway/build_krakend.py` menjadi `gateway/krakend.json`.
- **Alasan:** konfigurasi deklaratif, stateless, agregasi bawaan (pola BFF) tanpa service tambahan, topologi internal tersembunyi, CORS dan
  rate limit terpusat.
- **Konsekuensi:** gateway menjadi titik tunggal kegagalan pada lingkungan satu instance (mitigasi: stateless, dapat direplikasi);
  setiap rute baru harus didaftarkan eksplisit; header `Retry-After` pada respons 429 belum terverifikasi pada KrakenD CE; kunci pembatas IP
  rentan pada klien di balik NAT. gRPC catalog tidak lagi dibuka ke host (override sementara `docker-compose.grpc-debug.yml`).

## Keterbatasan yang kami sadari
- `JWT_SECRET`, `INTERNAL_TOKEN`, dan akun RabbitMQ masih sama untuk semua service.
- REST internal masih dipakai untuk perubahan status order oleh Payment/Logistics (kopling dua arah dengan Order);
  kandidat diganti event `PaymentCompleted`/`ShipmentDelivered`.
- Voucher pada checkout masih bernilai tetap; integrasi Promotion belum dilakukan.
- Pembatalan order mengabaikan galat release stok (REST) sehingga stok bisa tertahan bila Inventory mati; kandidat diganti event `OrderCancelled`.
- Gateway berjalan satu instance dan pembatasan laju memakai kunci IP (lihat ADR-008).
- Nomor order/faktur dibentuk dari waktu dengan kolom UNIQUE sehingga berisiko bentrok pada checkout dalam detik yang sama.
