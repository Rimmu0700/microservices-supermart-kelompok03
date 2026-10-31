# Architecture Decision Records – Kelompok 03

## ADR-001: PostgreSQL 15 untuk Order Service

**Keputusan.** Order Service memakai PostgreSQL 15 (`postgres:15-alpine`), port host 5434.

**Alasan.**
- Domain order menuntut transaksi finansial yang ketat dan integritas relasional (orders 1—N order_items) → cocok dengan ACID dan constraint PostgreSQL.
- Nominal rupiah disimpan sebagai integer (`BIGINT`), bebas galat pembulatan floating-point.
- Image alpine kecil dan `pg_isready` memudahkan healthcheck.

**Alternatif ditolak.** MongoDB (kurang ketat untuk transaksi order), Redis (in-memory, untuk sesi/keranjang sementara).

**Konsekuensi.** Tidak ada foreign key lintas database; hubungan ke layanan lain hanya berupa ID string/UUID.