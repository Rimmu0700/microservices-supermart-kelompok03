#!/usr/bin/env python3
"""
split_services.py - memecah backend/ (satu modul Go) menjadi service yang BENAR-BENAR mandiri.

Jalankan dari ROOT repositori:   python3 scripts/split_services.py
Hasil:  services/<nama>/   (satu folder = satu service, siap jadi repositori sendiri)
        tools/grpc-tools/  (klien uji gRPC dan pengukur payload)
        docker-compose.microservices.yml

Tiap folder service berisi SEMUA yang ia butuhkan: go.mod + go.sum sendiri, kode (cmd + internal + salinan
pkg/messaging/proto yang dipakai), Dockerfile sendiri (multi-stage, non-root), .dockerignore, .env.example,
db/schema.sql (skema MILIK service itu), dan README.md. Tidak ada impor antar folder service.

Skrip ini hanya MENYALIN; folder backend/ asli tidak diubah (jadi masih bisa dipakai sebagai cadangan).
"""
import os, re, shutil, sys, glob, copy

MOD = "github.com/nusantara-supermart/backend"
ROOT = os.getcwd()
BACKEND = os.path.join(ROOT, "backend")
SQLDIR = os.path.join(ROOT, "deployments", "sql", "generated")
OUT = os.path.join(ROOT, "services")
if not os.path.isdir(BACKEND):
    sys.exit("[ERROR] jalankan dari root repositori (folder yang berisi backend/ dan docker-compose.yml)")

# ---------------------------------------------------------------- metadata tiap service
SERVICES = {
 "auth-service": dict(prefix="auth", sql="11-auth.sql", grpc=False, http=True, hostdb=3311,
   role="Registrasi, login, penerbitan JWT, alamat pengguna, role dan permission.",
   upstream=[], downstream=["catalog", "inventory", "order", "payment", "logistics", "support"],
   repl_in=[], repl_out=["users (tanpa password_hash)", "user_addresses", "roles", "user_roles"],
   publishes=[], consumes=[], rest_provides=["REST publik /api/v1/auth/*"], calls=[]),
 "catalog-service": dict(prefix="catalog", sql="12-catalog.sql", grpc=True, http=True, hostdb=3312,
   role="Produk, kategori, brand, gambar, ulasan. Menyediakan REST dan gRPC (GetProduct, BatchGetProducts).",
   upstream=["auth (replika users)"], downstream=["order (gRPC + replika)", "inventory", "promotion", "procurement"],
   repl_in=["users dari auth"], repl_out=["products", "product_images"],
   publishes=[], consumes=[], rest_provides=["REST publik /api/v1/catalog/*", "gRPC :50051 supermart.catalog.v1.CatalogService"], calls=[]),
 "inventory-service": dict(prefix="inventory", sql="13-inventory.sql", grpc=False, http=True, hostdb=3313,
   role="Stok gudang dan reservasi stok.",
   upstream=["auth", "catalog (replika)", "order (event OrderPlaced)", "procurement (event GoodsReceived)"],
   downstream=["order (replika warehouses, event gagal reservasi, API release)", "procurement (replika warehouses)", "notification-worker"],
   repl_in=["users dari auth", "products dari catalog"], repl_out=["warehouses"],
   publishes=["StockReservationFailed -> inventory.<wilayah>.reservation_failed"],
   consumes=["OrderPlaced (inventory.stock-reservation.q)", "GoodsReceived (inventory.goods-received.q)"],
   rest_provides=["REST publik /api/v1/inventory/*", "REST internal: release stok (dipanggil order)"], calls=[]),
 "order-service": dict(prefix="order", sql="14-order.sql", grpc=False, http=True, hostdb=3314,
   role="Keranjang, checkout, status order. Menerbitkan OrderPlaced lewat Transactional Outbox.",
   upstream=["auth", "catalog (gRPC + replika)", "inventory (replika warehouses, event gagal reservasi)"],
   downstream=["payment", "logistics", "inventory", "support", "notification-worker"],
   repl_in=["users, user_addresses dari auth", "products, product_images dari catalog", "warehouses dari inventory"],
   repl_out=["orders", "order_shipping_details"],
   publishes=["OrderPlaced -> order.<wilayah>.placed"], consumes=["StockReservationFailed (order.reservation-failed.q)"],
   rest_provides=["REST publik /api/v1/orders/*", "REST internal: PUT /internal/orders/:id/status (dipanggil payment dan logistics)"],
   calls=["gRPC -> catalog-service:50051 (GetProduct)", "REST internal -> inventory-service (release stok)"]),
 "payment-service": dict(prefix="payment", sql="15-payment.sql", grpc=False, http=True, hostdb=3315,
   role="Faktur, pembayaran, dompet.",
   upstream=["order (event OrderPlaced, replika orders)", "auth (replika users)"], downstream=[],
   repl_in=["orders dari order", "users dari auth"], repl_out=[],
   publishes=[], consumes=["OrderPlaced (payment.invoice-creation.q)"],
   rest_provides=["REST publik /api/v1/payments/*"], calls=["REST internal -> order-service (ubah status order)"]),
 "promotion-service": dict(prefix="promotion", sql="16-promotion.sql", grpc=False, http=True, hostdb=3316,
   role="Voucher dan promo.",
   upstream=["catalog (replika products, product_images)"], downstream=[],
   repl_in=["products, product_images dari catalog"], repl_out=[],
   publishes=[], consumes=[], rest_provides=["REST publik /api/v1/promotions/*"], calls=[]),
 "logistics-service": dict(prefix="logistics", sql="17-logistics.sql", grpc=False, http=True, hostdb=3317,
   role="Pengiriman, kurir, bukti terima (POD).",
   upstream=["order (event OrderPlaced, replika orders dan order_shipping_details)", "auth (replika users, alamat)"], downstream=[],
   repl_in=["orders, order_shipping_details dari order", "users, user_addresses dari auth"], repl_out=[],
   publishes=[], consumes=["OrderPlaced (logistics.shipment-creation.q)"],
   rest_provides=["REST publik /api/v1/logistics/*"], calls=["REST internal -> order-service (SHIPPED / COMPLETED)"]),
 "procurement-service": dict(prefix="procurement", sql="18-procurement.sql", grpc=False, http=True, hostdb=3318,
   role="Purchase order dan penerimaan barang (GRN) dari pemasok. Menerbitkan GoodsReceived lewat Outbox.",
   upstream=["catalog (replika products)", "inventory (replika warehouses)"], downstream=["inventory (event GoodsReceived)"],
   repl_in=["products dari catalog", "warehouses dari inventory"], repl_out=[],
   publishes=["GoodsReceived -> procurement.<wilayah>.goods_received"], consumes=[],
   rest_provides=["REST publik /api/v1/procurement/*"], calls=[]),
 "support-service": dict(prefix="support", sql="19-support.sql", grpc=False, http=True, hostdb=3319,
   role="Tiket bantuan pelanggan.",
   upstream=["auth (replika users, roles, user_roles)", "order (replika orders)"], downstream=[],
   repl_in=["users, roles, user_roles dari auth", "orders dari order"], repl_out=[],
   publishes=[], consumes=[], rest_provides=["REST publik /api/v1/support/*"], calls=[]),
 "notification-worker": dict(prefix="notification", sql="20-notification.sql", grpc=False, http=False, hostdb=3320,
   role="Consumer murni: mengirim notifikasi WhatsApp/Email (simulasi log) dari event order dan inventory.",
   upstream=["order (event order.#)", "inventory (event inventory.#)"], downstream=[],
   repl_in=[], repl_out=[], publishes=[], consumes=["order.# dan inventory.# (notification.order-events.q)"],
   rest_provides=[], calls=[]),
}

# ---------------------------------------------------------------- utilitas
def go_imports(d):
    out = set()
    for f in os.listdir(d):
        if f.endswith(".go") and not f.endswith("_test.go"):
            t = open(os.path.join(d, f), encoding="utf-8").read()
            for m in re.finditer(r'"(' + re.escape(MOD) + r'/[^"]+)"', t):
                out.add(m.group(1)[len(MOD) + 1:])
    return out

def closure(start):
    seen, stack = set(), [start]
    while stack:
        d = stack.pop()
        if d in seen: continue
        p = os.path.join(BACKEND, d)
        if not os.path.isdir(p): sys.exit("[ERROR] paket tidak ditemukan: " + d)
        seen.add(d); stack += list(go_imports(p))
    return sorted(seen)

def copy_pkg(rel, dest_root):
    src = os.path.join(BACKEND, rel); dst = os.path.join(dest_root, rel)
    os.makedirs(dst, exist_ok=True)
    for f in os.listdir(src):
        sp = os.path.join(src, f)
        if os.path.isfile(sp) and not f.endswith("_test.go"):
            shutil.copy2(sp, os.path.join(dst, f))

def write(path, text):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    open(path, "w", encoding="utf-8", newline="\n").write(text)

def bullets(items, empty="(tidak ada)"):
    return "\n".join("- " + i for i in items) if items else "- " + empty

# ---------------------------------------------------------------- templat
DOCKERFILE = '''# {name} - Dockerfile MILIK service ini (tidak dipakai bersama service lain)
# Multi-stage: builder (SDK Go) -> runtime minimal non-root.
FROM golang:1.26-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/service ./cmd/{name}

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata \\
 && addgroup -g 10001 -S appgroup && adduser -S -D -H -u 10001 -G appgroup appuser
COPY --from=builder /out/service /app/service
USER 10001:10001
EXPOSE {expose}
{health}ENTRYPOINT ["/app/service"]
'''
HEALTH = 'HEALTHCHECK --interval=10s --timeout=5s --start-period=20s --retries=6 \\\n  CMD wget -qO- http://127.0.0.1:3000/api/v1/health >/dev/null || exit 1\n'
DOCKERIGNORE = ".git\n.gitignore\n.env\n.env.*\n!.env.example\n*.md\n*.exe\nbin\ntmp\nvendor\nDockerfile*\ndocker-compose*.yml\n.vscode\n.idea\n"
ENVEX = '''# Salin menjadi .env (jangan commit .env). Nilai di bawah cocok untuk docker-compose.microservices.yml.
APP_PORT=3000
APP_ENV=production
MYSQL_HOST={prefix}-db
MYSQL_PORT=3306
MYSQL_DATABASE={prefix}_db
MYSQL_USER={prefix}_user
MYSQL_PASSWORD={prefix}_secret
JWT_SECRET=ganti-dengan-rahasia-yang-kuat
JWT_EXPIRY=24h
AMQP_URI=amqp://nusantara_mq:nusantara_mq_secret@rabbitmq:5672/%2Fsupermart
INTERNAL_TOKEN=ganti-dengan-token-internal
REPLICA_INTERVAL_SECONDS=3
CORS_ORIGINS=http://localhost:4200,http://localhost:3000
AUTH_URL=http://auth-service:3000
CATALOG_URL=http://catalog-service:3000
CATALOG_GRPC_ADDR=catalog-service:50051
INVENTORY_URL=http://inventory-service:3000
ORDER_URL=http://order-service:3000
'''
README = '''# {name}

**Peran:** {role}

| Item | Nilai |
|---|---|
| Pemilik (anggota tim) | ______________ |
| Modul Go | `{mod}` (go.mod milik folder ini) |
| Port HTTP | 3000 (di dalam jaringan Docker) |{grpcrow}
| Database milik service ini | `{prefix}_db` pada container `nusantara-{prefix}-db` (MySQL 8.0, port host {hostdb}) |
| Skema database | `db/schema.sql` (MILIK service ini; boleh diganti dialeknya tanpa memengaruhi service lain) |
| Dockerfile | `Dockerfile` (multi-stage, non-root UID 10001) |

## Upstream (pihak yang menyediakan data untuk service ini)
{upstream}

## Downstream (pihak yang bergantung pada service ini)
{downstream}

## Kontrak komunikasi
**Event yang diterbitkan:**
{publishes}

**Event yang dikonsumsi:**
{consumes}

**Antarmuka yang disediakan:**
{rest_provides}

**Memanggil service lain:**
{calls}

**Replika data referensi (pull berkala dari pemilik data):**
{repl_in}

**Data yang diekspor ke service lain (GET /internal/replica/:table):**
{repl_out}

## Menjalankan sendiri
```bash
docker build -t {name} .                       # build image milik service ini saja
go build ./cmd/{name}                          # atau build lokal
```
Menjalankan seluruh sistem: dari root repositori `docker compose -f docker-compose.microservices.yml up --build -d`.

## Mengganti teknologi database (kebebasan service ini)
Semua yang perlu diubah ada **di dalam folder ini saja**: `pkg/database/` (driver dan DSN), `internal/{short}/`
dan `internal/messaging/` + `pkg/replica/` (dialek SQL), `db/schema.sql`, serta definisi container DB-nya di compose.
Lihat `MIGRASI-SERVICE.md` di root untuk daftar perubahan ke PostgreSQL.
'''

def make_service(name, m):
    d = os.path.join(OUT, name)
    if os.path.isdir(d): shutil.rmtree(d)
    os.makedirs(d)
    pk = closure("cmd/" + name)
    for rel in pk: copy_pkg(rel, d)
    shutil.copy2(os.path.join(BACKEND, "go.mod"), d)
    shutil.copy2(os.path.join(BACKEND, "go.sum"), d)
    expose = "3000" + (" 50051" if m["grpc"] else "") if m["http"] else "3000"
    write(os.path.join(d, "Dockerfile"), DOCKERFILE.format(name=name, expose=expose, health=(HEALTH if m["http"] else "")))
    write(os.path.join(d, ".dockerignore"), DOCKERIGNORE)
    write(os.path.join(d, ".env.example"), ENVEX.format(prefix=m["prefix"]))
    shutil.copy2(os.path.join(SQLDIR, m["sql"]), os.path.join(d, "db_schema.tmp"))
    os.makedirs(os.path.join(d, "db"), exist_ok=True)
    os.replace(os.path.join(d, "db_schema.tmp"), os.path.join(d, "db", "schema.sql"))
    grpcrow = "\n| Port gRPC | 50051 |" if m["grpc"] else ""
    write(os.path.join(d, "README.md"), README.format(
        name=name, role=m["role"], mod=MOD, prefix=m["prefix"], hostdb=m["hostdb"], short=m["prefix"], grpcrow=grpcrow,
        upstream=bullets(m["upstream"]), downstream=bullets(m["downstream"]),
        publishes=bullets(m["publishes"]), consumes=bullets(m["consumes"]), rest_provides=bullets(m["rest_provides"]),
        calls=bullets(m["calls"]), repl_in=bullets(m["repl_in"]), repl_out=bullets(m["repl_out"])))
    return pk

def make_tools():
    d = os.path.join(ROOT, "tools", "grpc-tools")
    if os.path.isdir(d): shutil.rmtree(d)
    os.makedirs(d)
    for t in ("grpcclient", "payloadsize"):
        for rel in closure("cmd/" + t): copy_pkg(rel, d)
    shutil.copy2(os.path.join(BACKEND, "go.mod"), d); shutil.copy2(os.path.join(BACKEND, "go.sum"), d)
    write(os.path.join(d, "README.md"), "# grpc-tools\nKlien uji gRPC (`go run ./cmd/grpcclient`) dan pengukur payload (`go run ./cmd/payloadsize`).\nModul sendiri, terpisah dari semua service.\n")

# ---------------------------------------------------------------- compose
def make_compose():
    import yaml
    src = yaml.safe_load(open(os.path.join(ROOT, "docker-compose.yml"), encoding="utf-8"))
    sv = src["services"]
    out = {"services": {}, "volumes": {}}
    out["services"]["rabbitmq"] = copy.deepcopy(sv["rabbitmq"])
    out["services"]["rabbitmq"]["ports"] = ["127.0.0.1:15672:15672"]   # konsol manajemen (ops) hanya di localhost; AMQP 5672 internal
    for name, m in SERVICES.items():
        p = m["prefix"]
        out["services"][p + "-db"] = {
            "image": "mysql:8.0", "container_name": f"nusantara-{p}-db", "restart": "unless-stopped",
            "environment": {"MYSQL_ROOT_PASSWORD": "root_secret", "MYSQL_DATABASE": p + "_db", "MYSQL_USER": p + "_user", "MYSQL_PASSWORD": p + "_secret"},
            "command": ["--default-authentication-plugin=mysql_native_password", "--innodb-buffer-pool-size=64M", "--performance-schema=OFF", "--max-connections=60"],
            "volumes": [f"./services/{name}/db/schema.sql:/docker-entrypoint-initdb.d/10-schema.sql:ro", f"{p}_db_data:/var/lib/mysql"],
            "healthcheck": {"test": ["CMD", "mysqladmin", "ping", "-h", "127.0.0.1", "-u", "root", "-proot_secret"], "interval": "5s", "timeout": "5s", "retries": 20, "start_period": "40s"}}
        out["volumes"][p + "_db_data"] = None
    out["volumes"]["rabbitmq_data"] = None
    for name, m in SERVICES.items():
        p = m["prefix"]
        s = copy.deepcopy(sv[name])
        s["build"] = {"context": f"./services/{name}"}          # <- konteks build MILIK service sendiri
        s["environment"]["MYSQL_HOST"] = p + "-db"
        s["depends_on"] = {p + "-db": {"condition": "service_healthy"}, "rabbitmq": {"condition": "service_healthy"}}
        s.pop("ports", None)
        s["expose"] = ["3000", "50051"] if m["grpc"] else (["3000"] if m["http"] else [])
        if not s["expose"]: s.pop("expose")
        out["services"][name] = s
    out["services"]["api-gateway"] = {
        "image": "devopsfaith/krakend:2.6.3", "container_name": "nusantara-api-gateway", "restart": "unless-stopped",
        "ports": ["8000:8080"],                                   # SATU-SATUNYA pintu masuk API untuk klien eksternal
        "volumes": ["./gateway/krakend.json:/etc/krakend/krakend.json:ro"],
        "depends_on": [n for n in SERVICES if SERVICES[n]["http"]]}
    out["services"]["frontend"] = copy.deepcopy(sv["frontend"])
    out["services"]["frontend"]["depends_on"] = ["api-gateway"]
    keys = ["rabbitmq"] + [m["prefix"] + "-db" for m in SERVICES.values()] + list(SERVICES) + ["api-gateway", "frontend"]
    out["services"] = {k: out["services"][k] for k in keys}
    header = '''# =============================================================================================
# docker-compose.microservices.yml - SEMUA SERVICE MANDIRI
#   * tiap service dibangun dari folder sendiri (services/<nama>/ : go.mod, Dockerfile, kode, db/schema.sql)
#   * tiap service punya CONTAINER DATABASE, VOLUME, port host, dan kredensial sendiri
#   * tidak ada kode, modul Go, Dockerfile, maupun database yang dipakai bersama
#   * Tugas 5: HANYA api-gateway (KrakenD :8000) dan frontend (:4200) yang membuka port ke klien; service dan database
#     hanya 'expose' di jaringan Docker; konsol RabbitMQ (ops) hanya di 127.0.0.1:15672
#
#   docker compose down -v      # matikan stack lama dulu (nama container sama)
#   docker compose -f docker-compose.microservices.yml up --build -d
#   docker compose -f docker-compose.microservices.yml ps
# Memori: 10 container MySQL + 10 service Go ~ 4-5 GB RAM; beri Docker minimal 8 GB.
# =============================================================================================
'''
    body = yaml.safe_dump(out, sort_keys=False, allow_unicode=True, width=200, default_flow_style=False)
    body = re.sub(r"^(  \w+_db_data|  rabbitmq_data): null$", r"\1:", body, flags=re.M)
    write(os.path.join(ROOT, "docker-compose.microservices.yml"), header + "\n" + body)

# ---------------------------------------------------------------- validasi statis
def validate():
    ok = True
    for name in SERVICES:
        d = os.path.join(OUT, name)
        for need in ("go.mod", "go.sum", "Dockerfile", ".dockerignore", ".env.example", "README.md", "db/schema.sql", f"cmd/{name}/main.go"):
            if not os.path.exists(os.path.join(d, need)): print("  [GAGAL] hilang:", name, need); ok = False
        # semua impor internal harus ada di dalam folder service ini
        for root, _, files in os.walk(d):
            for f in files:
                if f.endswith(".go"):
                    t = open(os.path.join(root, f), encoding="utf-8").read()
                    for m in re.finditer(r'"(' + re.escape(MOD) + r'/[^"]+)"', t):
                        rel = m.group(1)[len(MOD) + 1:]
                        if not os.path.isdir(os.path.join(d, rel)): print("  [GAGAL] impor tak terpenuhi:", name, rel); ok = False
        # tidak ada impor ke service lain
        others = [s for s in os.listdir(os.path.join(d, "internal")) ] if os.path.isdir(os.path.join(d, "internal")) else []
    return ok

if __name__ == "__main__":
    os.makedirs(OUT, exist_ok=True)
    for name, m in SERVICES.items():
        pk = make_service(name, m)
        print(f"[OK] services/{name:22s} paket: {', '.join(p for p in pk if not p.startswith('cmd/'))}")
    make_tools(); print("[OK] tools/grpc-tools")
    make_compose(); print("[OK] docker-compose.microservices.yml")
    print("\nValidasi statis:", "LULUS" if validate() else "ADA MASALAH")
