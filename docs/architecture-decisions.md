# Architecture Decision Records — Nusantara SuperMart (Kelompok 03)

## ADR-Identity-01: Database terisolasi MySQL 8 untuk Identity Service

**Status:** Diterima
**Domain:** Identity, Akun & RBAC (Kluster 1 — 12 tabel)
**Penanggung jawab:** Maria Dwi Rosari Nahak

### Konteks
Pada monolith, seluruh 120 tabel berada di satu database `nusantara_db`. Tabel `users`
menjadi referensi `user_id` bagi hampir semua kluster lain (Order, Payment, Support,
Logistics, Inventory). Kopling ini membuat perubahan skema identitas berisiko
merusak domain lain. Pola *Database-per-Service* mengharuskan setiap service
memiliki penyimpanannya sendiri.

### Keputusan
1. Identity Service memakai **MySQL 8.0** dengan database `identity_service_db`,
   dijalankan sebagai container `supermart-identity-db` di jaringan `supermart-isolated-net`.
2. Port host **3307** (port 3306 sudah dipakai MySQL monolith `nusantara_db`,
   sehingga monolith dan service baru bisa berjalan bersamaan selama migrasi).
3. Skema berisi 12 tabel Kluster 1: `users`, `user_profiles`, `user_addresses`,
   `user_kyc_documents`, `roles`, `permissions`, `role_permissions`, `user_roles`,
   `auth_tokens`, `login_histories`, `password_resets`, `two_factor_auths`.
4. Skema dan data awal dimuat otomatis lewat `docker-entrypoint-initdb.d`
   (`init-scripts/identity/01-schema.sql`, `02-seed.sql`).

### Alasan memilih MySQL (Polyglot Persistence)
| Kebutuhan domain Identity | Mengapa MySQL cocok |
|---|---|
| Data sangat relasional (user ↔ role ↔ permission, many-to-many) | Foreign key & JOIN native |
| Konsistensi kuat (email unik, token unik, status akun) | Transaksi ACID InnoDB, constraint `UNIQUE` |
| Skema stabil, jarang berubah | Skema ketat lebih aman daripada skema bebas |
| Kompatibel dengan kode monolith (`backend/internal/auth` memakai sqlx + MySQL) | Migrasi tanpa menulis ulang query |

Domain lain memilih DBMS sesuai karakteristiknya: Catalog → MongoDB (atribut produk
fleksibel), Order → PostgreSQL (transaksi pesanan).

### Konsekuensi
- **Tidak ada FOREIGN KEY lintas database.** Service lain menyimpan `user_id` sebagai
  referensi logis saja, dan memvalidasi identitas lewat **JWT** yang diterbitkan
  Identity Service (klaim `user_id`, `email`, `role`; HS256), bukan JOIN ke tabel `users`.
- Data yang dibutuhkan service lain (misal nama/alamat pada pesanan) disalin sebagai
  *snapshot* saat transaksi, atau diambil melalui API Identity Service.
- Alamat pengguna (`user_addresses`) tetap milik Identity; Order menyimpan salinan alamat
  pengiriman pada saat checkout.

### Cara menjalankan & verifikasi
```bash
cd deployments/docker
docker compose up -d identity-db
docker exec -it supermart-identity-db \
  mysql -uidentity_admin -pidentity_secret_pass identity_service_db -e "SHOW TABLES;"
```
Hasil yang diharapkan: 12 tabel, 5 role, 9 permission, dan akun
`admin@nusantara.co.id` (password `password123`) dengan role `SUPER_ADMIN`.
