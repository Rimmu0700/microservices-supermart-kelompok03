# Tugas 5 - API Gateway (KrakenD): Catatan Teknis untuk Laporan

Dokumen kerja, bukan laporan. Isinya fakta dari implementasi kelompok untuk memudahkan penulisan empat bagian laporan
(Bagian I-IV). Semua angka hasil pengujian **harus diambil dari mesin sendiri** (skrip di `scripts/gateway/`).

## 0. Ringkasan implementasi
| Item | Nilai |
|---|---|
| Gateway | KrakenD `devopsfaith/krakend:2.6.3`, container `nusantara-api-gateway`, port host **8000** |
| Konfigurasi | `gateway/krakend.json` (HASIL GABUNGAN dari `gateway/base.json` + `gateway/routes/*.json`) |
| Rute pass-through | 101 rute (seluruh REST 9 service) + 2 alias path rewriting |
| Agregasi paralel | `/api/v1/storefront` (catalog + promotion), `/api/v1/order-summary/{order_id}` (order + payment + logistics), `/api/v1/health` (9 service) |
| Rate limiting | login 10/menit, register 5/menit, checkout 10/menit, per IP, algoritma token bucket |
| Isolasi | layanan dan database hanya `expose`; port host hanya gateway `:8000`, frontend `:4200`, dan konsol RabbitMQ `127.0.0.1:15672` |
| Frontend | `frontend/nginx.conf` meneruskan `/api/` ke `api-gateway:8080` (menggantikan nginx `gateway` lama) |

Menjalankan: `docker compose -f docker-compose.yml up --build -d`, lalu `bash scripts/gateway/01-...` sampai `04-...`.
Mengubah rute: edit `gateway/routes/<domain>.json`, jalankan `python3 gateway/build_krakend.py`, lalu `docker compose restart api-gateway`.
Validasi memakai KrakenD sendiri: `docker run --rm -v "$PWD/gateway/krakend.json:/etc/krakend/krakend.json:ro" devopsfaith/krakend:2.6.3 check -c /etc/krakend/krakend.json`.

## Bagian I - Arsitektur edge dan topologi
### 1. Audit eksposur sebelum gateway (Milestone 1)
`docker-compose.yml` Milestone 1 membuka ke host: MySQL `3306`, RabbitMQ `5672` dan `15672`, gRPC catalog `50051`, nginx gateway `3000`,
frontend `4200`; pada varian satu-DB-per-service, database memakai port host `3311`-`3320`.
Kelemahan yang dapat dianalisis:
- **Keamanan jaringan:** permukaan serangan lebar; basis data dan broker dapat dijangkau langsung dari luar container; gRPC internal
  terbuka tanpa autentikasi gateway.
- **CORS:** bila klien memanggil banyak port, tiap service harus mengatur CORS sendiri (duplikasi kebijakan dan risiko tidak konsisten);
  sekarang CORS dipusatkan di gateway (`security/cors` pada `gateway/base.json`).
- **Client-to-service coupling:** klien harus tahu topologi (port, nama service); perubahan internal memecahkan klien. Gateway
  menyembunyikan topologi di balik jalur publik yang stabil.
- **Chatty round-trip:** tampilan seperti ringkasan pesanan memerlukan beberapa panggilan berurutan dari klien.

### 2. North-South vs East-West
```
 Klien eksternal (browser / Postman / aplikasi mobile)
        |  North-South (HTTPS/REST+JSON, satu pintu)
        v
 [ frontend nginx :4200 ] --/api/--> [ API GATEWAY KrakenD :8000 ]   <- routing, path rewriting, agregasi, rate limit, CORS
                                              |   jaringan Docker privat (tanpa port host)
        +---------+---------+---------+------+------+---------+---------+---------+
        v         v         v         v             v         v         v         v
      auth     catalog   inventory   order        payment  logistics procurement  support (+ promotion)
        |         ^  gRPC (order -> catalog :50051, East-West sinkron)
        |         +---------------------------------------------+
        +---- event RabbitMQ (OrderPlaced, StockReservationFailed, GoodsReceived; East-West asinkron) ----+
        +---- REST internal bertoken (payment/logistics -> order) ----+
        +---- DB per service (container sendiri, tanpa port host) ----+
```
- **North-South:** satu-satunya jalur klien eksternal; pintu tunggal, stateless, REST/JSON, kebijakan lintas-layanan (CORS, rate limit).
- **East-West:** komunikasi privat antarlayanan di jaringan Docker lewat gRPC (kueri sinkron), RabbitMQ (event asinkron), dan REST internal
  bertoken. Tidak melewati gateway dan tidak terjangkau dari host.

### 3. Justifikasi teknologi, anti-pattern, dan SPOF
- **Mengapa KrakenD:** konfigurasi deklaratif (JSON, tanpa kode), stateless, agregasi paralel bawaan (fan-out + merge + filter) sehingga
  pola BFF tidak butuh service tambahan; rate limit bawaan. Kong OSS/Traefik/Envoy kuat pada routing dan rate limit tetapi tidak
  menyediakan fan-out komposisi respons secara deklaratif; Custom Go proxy fleksibel tetapi memindahkan beban pemeliharaan ke tim.
- **Menghindari Gateway Anti-Pattern:** gateway hanya melakukan routing, path rewriting, agregasi baca-saja, penyaringan atribut respons,
  rate limit, dan CORS. Tidak ada aturan bisnis (harga, stok, status order) dan tidak ada transformasi yang mengubah makna data.
  Rute pass-through memakai `no-op` sehingga status dan isi respons backend diteruskan apa adanya. Validasi JWT dan otorisasi peran
  tetap dikerjakan service (pertahanan berlapis), gateway hanya meneruskan header `Authorization`.
- **Mitigasi SPOF:** gateway stateless sehingga dapat direplikasi di belakang load balancer (mis. beberapa instance + LB, atau Kubernetes
  Deployment); `restart: unless-stopped`; di Compose satu host, kegagalan gateway tidak menghentikan komunikasi East-West
  (gRPC/event tetap jalan). Keterbatasan: pada lingkungan lab ini hanya satu instance.

## Bagian II - Routing, agregasi, degradasi
- **Routing + path rewriting:** 101 rute mencerminkan jalur service (`/api/v1/<domain>/...`) dengan `no-op`, dan dua alias menunjukkan
  penulisan ulang jalur: `GET /api/v1/products/{id}` menjadi `/api/v1/catalog/products/{id}` dan `GET /api/v1/invoices/{order_id}`
  menjadi `/api/v1/payments/invoices/order/{order_id}`. Berkas: `gateway/routes/*.json`.
- **Agregasi (BFF):** `order-summary` memanggil 3 backend paralel, memakai `target: "data"` untuk membuka amplop `{success,data}`, dan
  `deny` untuk membuang atribut yang tidak dibutuhkan klien (`histories`, `notes`, `shipping` pada order; `tracking_logs`, `proof` pada
  pengiriman). `storefront` menggabungkan produk, kategori (catalog) dan flash sale (promotion) tanpa login.
- **Graceful degradation:** bila satu backend gagal, KrakenD tetap membalas HTTP 200 dengan data backend yang hidup dan menandai
  respons dengan header `X-Krakend-Completed: false`. Dua kondisi diuji (skrip `02`): service **mati** (koneksi ditolak, jawaban cepat)
  dan service **macet** (`docker pause`, gateway menunggu sampai `timeout` endpoint 2500 ms lalu menjawab parsial).
- **Isolasi jaringan:** skrip `01` membuktikan akses lewat `:8000` berhasil, sedangkan `localhost:3000/3306/5672/50051/3311/3314` ditolak.

## Bagian III - Rate limiting
- **Kebijakan:** `qos/ratelimit/router` pada tiga rute kritis, **per klien**: `client_max_rate` = 10 per `1m`, `client_capacity` = 10,
  `strategy: ip`. Login dipilih karena rawan brute force; register 5/menit; checkout 10/menit.
- **Algoritma: token bucket.** Ember berkapasitas 10 token, terisi 1 token tiap 6 detik. Burst hingga 10 permintaan langsung diterima,
  setelah itu laju rata-rata dibatasi 10/menit. Dibanding fixed window: tidak ada lonjakan ganda di batas jendela; dibanding sliding
  window: lebih murah dihitung tetapi mengizinkan burst sebesar kapasitas.
- **Kunci klien: alamat IP.** Risiko NAT: banyak pengguna di balik satu IP (kantor, kampus, operator seluler) berbagi satu ember sehingga satu
  pengguna dapat menghabiskan jatah yang lain. Pada arsitektur ini permintaan dari SPA lewat `frontend nginx` terlihat berasal dari container
  frontend; pastikan header `X-Forwarded-For` diperhitungkan, atau gunakan kunci berbasis header/JWT (mis. `strategy: header`, `key: X-API-Key`).
  Alternatif: batasi per akun pengguna setelah login dan per IP sebelum login.
- **Mengapa memakai `client_*` saja:** limiter tingkat endpoint (`max_rate`/`capacity`) menolak dengan **503**, sedangkan limiter per klien
  menolak dengan **429**. Agar hasil uji konsisten 429, hanya limiter per klien yang dipasang.
- **Catatan konfigurasi penting:** `client_capacity` bila tidak ditulis bernilai sangat kecil untuk laju per-menit (kapasitas bawaan dihitung
  dari laju per detik), sehingga permintaan kedua bisa langsung ditolak. Karena itu `client_capacity` dituliskan eksplisit.
- **Header Retry-After:** dokumentasi KrakenD CE tidak menyebut header `Retry-After`/`X-RateLimit-*` pada respons 429 limiter router;
  **belum terverifikasi** apakah ada. Skrip `03` mencetak header mentah dan memberi `[PERHATIAN]` bila tidak ada. Bila dosen mensyaratkan header:
  (a) sertakan bukti 429 dan jelaskan bahwa KrakenD CE tidak menambahkannya, atau (b) tambah proxy tipis di depan gateway yang menambah header, mis.
  pada nginx: `limit_req_status 429;` + `error_page 429 @rl;` + `location @rl { add_header Retry-After 6 always; return 429; }` (belum diuji).

## Bagian IV - Evaluasi kinerja
`bash scripts/gateway/04-ukur-latensi-overhead-dan-agregasi.sh` (N=20 percobaan + 3 pemanasan, dari dalam jaringan Docker memakai
container `curlimages/curl` agar tanpa NAT host) menghasilkan tabel: (1) overhead rute tunggal langsung vs via gateway,
(2) 3 request berurutan vs 1 endpoint agregasi, (3) latensi respons 429 saat beban berlebih. Hasil tersimpan di `hasil-demo/`.

## Catatan jujur
- Konfigurasi dan skrip **belum dijalankan terhadap KrakenD asli** oleh penyusun; jalankan `krakend check` dan skrip `01`-`04` lebih dulu.
- Contoh konfigurasi di modul memakai `capacity: 5` global dan tanpa `client_capacity`; konfigurasi kelompok sengaja berbeda (lihat Bagian III).
- Port host database (`3311`-`3320`, konvensi Praktikum 1) dan port AMQP `5672` dihapus pada compose ini agar memenuhi aturan isolasi Tugas 5;
  akses ke database memakai `docker exec`.
- gRPC catalog kini hanya internal; untuk klien/benchmark gRPC dari host pakai `docker-compose.grpc-debug.yml` (sementara, hanya localhost).
