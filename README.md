# PT Nusantara SuperMart Indonesia - Microservices (Kelompok 03)

Aplikasi toko swalayan yang dipecah dari *modular monolith* menjadi **9 microservice + 1 worker**. Setiap service punya
kode, modul Go, Dockerfile, skema, dan **database sendiri**; klien hanya masuk lewat **API Gateway (KrakenD)**.
README monolit lama diarsipkan di `docs/legacy/README-monolith-seed.md`.

## Arsitektur
```
 Browser -> frontend (nginx :4200) --/api/--> API Gateway KrakenD (:8000)        <- North-South (satu pintu)
                                                   | jaringan Docker privat (tanpa port host)
   auth  catalog  promotion  inventory  procurement  order  payment  logistics  support   notification-worker
    |       ^  gRPC (order -> catalog)         |        event RabbitMQ (OrderPlaced, GoodsReceived, ...)
    +-- DB per service (container MySQL sendiri) <- East-West: gRPC, RabbitMQ, REST internal bertoken
```

| Service | Peran | Database (container) | Pemilik |
|---|---|---|---|
| auth-service | login, JWT, alamat, role | auth-db | ____ |
| support-service | tiket bantuan | support-db | ____ |
| catalog-service | produk, kategori, ulasan; REST + gRPC | catalog-db | ____ |
| promotion-service | voucher, flash sale | promotion-db | ____ |
| inventory-service | stok dan reservasi | inventory-db | ____ |
| procurement-service | PO dan GRN (outbox) | procurement-db | ____ |
| notification-worker | notifikasi dari event | notification-db | ____ |
| order-service | keranjang, checkout (outbox) | order-db | ____ |
| payment-service | faktur, pembayaran, dompet | payment-db | ____ |
| logistics-service | pengiriman, kurir | logistics-db | ____ |

Detail upstream/downstream, event, replika data, dan kontrak tiap service ada di `services/<nama>/README.md`.
Keputusan arsitektur: `docs/architecture-decisions.md`. Catatan API Gateway: `docs/tugas5-api-gateway.md`.

## Menjalankan
```bash
cp .env.example .env                               # bila diperlukan
docker compose -f docker-compose.yml up --build -d
docker compose -f docker-compose.yml ps
```
(Setelah `scripts/finalize-microservices.sh --apply`, cukup `docker compose up --build -d`.)

| Alamat | Fungsi |
|---|---|
| http://localhost:4200 | Frontend Angular |
| http://localhost:8000/api/v1/health | API Gateway (agregasi health 9 service) |
| http://localhost:15672 | Konsol RabbitMQ (hanya localhost; user `nusantara_mq`, vhost `/supermart`) |

Service dan database **tidak** membuka port ke host (hanya `expose` di jaringan Docker). Untuk klien gRPC dari host pakai
`docker-compose.grpc-debug.yml` sementara.

## Struktur repositori
```
services/<nama>/   service mandiri: go.mod, Dockerfile, db/schema.sql, kode, README
gateway/           konfigurasi KrakenD (base.json + routes/<domain>.json -> krakend.json)
frontend/          Angular SPA + nginx
proto/             kontrak gRPC (sumber)
tools/grpc-tools/  klien uji gRPC
scripts/           demo/ (skrip demo), gateway/ (uji Tugas 5), audit/ (audit microservice), smoke-test.sh
docs/              keputusan arsitektur, catatan Tugas 5, arsip monolit (legacy/)
```

## Pengujian dan audit
```bash
bash scripts/smoke-test.sh                          # satu transaksi belanja end-to-end lewat gateway
python3 scripts/audit/audit_microservices.py        # audit otomatis kemandirian service
bash scripts/gateway/01-uji-routing-dan-isolasi-port.sh   # ... sampai 04 (Tugas 5)
bash scripts/demo/00-cek-prasyarat-sebelum-demo.sh        # skrip demo UTS (A-E, D1-D5)
```

## 👥 Daftar Akun Demo & Kredensial Pengujian

Semua akun pra-konfigurasi dalam `seed.sql` menggunakan kata sandi yang sama: **`password123`**

| Peran (Role) | Alamat Email | Kata Sandi | Portal Akses & Alur Uji Coba |
|---|---|---|---|
| **Super Admin** | `admin@nusantara-supermart.co.id` | `password123` | Dashboard Admin, Verifikasi KYC, Kelola Pengguna, Katalog, Voucher Promosi. |
| **Staf Gudang Jakarta** | `budi.gudang@nusantara-supermart.co.id` | `password123` | Pantau Stok Jakarta Hub, Lakukan Penyesuaian Stok, Mutasi Barang, Terima GRN. |
| **Staf Gudang Surabaya** | `eko.gudang@nusantara-supermart.co.id` | `password123` | Pantau Stok Surabaya Hub, Mutasi Barang ke Jakarta/Bali, Terima Pasokan GRN. |
| **Kurir Jakarta** | `kurir.jkt@nusantara-supermart.co.id` | `password123` | Antrean Kiriman Jakarta, Ubah ke In-Transit, Konfirmasi POD dengan Foto. |
| **Kurir Surabaya** | `kurir.sby@nusantara-supermart.co.id` | `password123` | Antrean Kiriman Surabaya, Ubah ke In-Transit, Konfirmasi POD dengan Foto. |
| **Agen CS** | `siti.cs@nusantara-supermart.co.id` | `password123` | Antrean Tiket Bantuan, Ambil Tiket (*Assign*), Kirim Balasan Pesan ke Pelanggan. |
| **Pelanggan (Siti)** | `siti.aminah@gmail.com` | `password123` | Belanja Katalog, Masukkan Keranjang, Checkout Alamat, Bayar Tagihan, Dompet. |
| **Pelanggan (Budi)** | `budi.santoso@yahoo.com` | `password123` | Belanja Katalog, Masukkan Voucher Promo, Buat Pesanan, Ajukan Tiket Bantuan CS. |

> 💡 **Fitur Praktis**: Halaman login (`/auth/login`) dilengkapi tombol **1-Click Test Login** untuk setiap akun di atas sehingga Anda tidak perlu mengetikkan email dan password berulang kali.

## Batasan yang kami sadari
Kode pustaka (`internal/messaging`, `pkg/`) disalin ke tiap service; `JWT_SECRET` dan `INTERNAL_TOKEN` masih sama untuk semua service;
semua database masih MySQL 8.0 (struktur mendukung pergantian per service, lihat `MIGRASI-SERVICE.md` bagian 4).
