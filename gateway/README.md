# gateway/ - API Gateway (KrakenD)

| Berkas | Pemilik | Fungsi |
|---|---|---|
| `base.json` | Anggota 1 | Pengaturan global (nama, port, timeout, CORS) |
| `routes/gateway.json` | Anggota 1 | Agregasi `/api/v1/health` |
| `routes/identity.json` | Anggota 1 | Rute auth dan support + rate limit login/register |
| `routes/catalog.json` | Anggota 2 | Rute catalog dan promotion, alias `/products/{id}`, agregasi `/storefront` |
| `routes/inventory.json` | Anggota 3 | Rute inventory dan procurement |
| `routes/order.json` | Anggota 4 | Rute order, payment, logistics, alias `/invoices/{order_id}`, agregasi `/order-summary/{order_id}`, rate limit checkout |
| `build_krakend.py` | Anggota 1 | Menggabungkan semua di atas menjadi `krakend.json` dan memeriksa konsistensi |
| `krakend.json` | Anggota 1 | HASIL GABUNGAN yang dipasang ke container |
| `tools/gen_routes_from_routers.py` | Anggota 1 | Pembuat kerangka awal rute dari berkas router Go (dipakai sekali) |

Alur kerja: ubah `routes/<domain>.json` milikmu, lalu `python3 gateway/build_krakend.py`. Setelah semua PR di-merge, **Anggota 1**
menjalankan skrip itu sekali lagi dan meng-commit `krakend.json` agar tidak ada konflik merge.
