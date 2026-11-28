#!/usr/bin/env bash
# =====================================================================================
# NAMA       : 20-E-bukti-isolasi-data-database-per-service.sh
# KATEGORI   : PEMBUKTIAN  (sistem sehat)   -> Bagian E (Skor Kelompok: Isolasi Database)
# FUNGSI     : Membuktikan database-per-service: database terpisah, user per database,
#              TIDAK ADA foreign key lintas database, snapshot harga historis di order_items,
#              dan tabel idempotensi/outbox tersedia di tiap database service.
# DIJALANKAN : bash scripts/demo/20-E-bukti-isolasi-data-database-per-service.sh
# =====================================================================================
# ======================= BAGIAN BERSAMA (tidak perlu diubah) =======================
set -uo pipefail
export MSYS_NO_PATHCONV=1   # Git Bash (Windows): cegah "/supermart" diubah menjadi path Windows

# --- cari folder proyek (yang berisi docker-compose.yml) ---
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
while [ ! -f "$ROOT/docker-compose.yml" ] && [ "$ROOT" != "/" ]; do ROOT="$(dirname "$ROOT")"; done
[ -f "$ROOT/docker-compose.yml" ] || { echo "[ERROR] docker-compose.yml tidak ditemukan. Letakkan skrip ini di dalam folder proyek (mis. scripts/demo/)."; exit 1; }
cd "$ROOT"

# --- konfigurasi (bisa diganti lewat environment, nilai bawaan sesuai repo) ---
BASE="${BASE:-http://localhost:8000/api/v1}"   # API Gateway KrakenD (Tugas 5)
EMAIL="${EMAIL:-wayan.putra@gmail.com}"
PASSWORD="${PASSWORD:-password123}"
PRODUCT_ID="${PRODUCT_ID:-p0000001-0000-0000-0000-000000000001}"
ADDRESS_ID="${ADDRESS_ID:-a0000001-0000-0000-0000-000000000001}"
WAREHOUSE_ID="${WAREHOUSE_ID:-w0000001-0000-0000-0000-000000000001}"
PAYMENT_METHOD_ID="${PAYMENT_METHOD_ID:-pm000001-0000-0000-0000-000000000001}"
MQ_CONTAINER="${MQ_CONTAINER:-nusantara-rabbitmq}"
MYSQL_CONTAINER="${MYSQL_CONTAINER:-nusantara-mysql}"
MQ_USER="${MQ_USER:-nusantara_mq}"
MQ_PASS="${MQ_PASS:-nusantara_mq_secret}"
MQ_VHOST="${MQ_VHOST:-/supermart}"
MQ_API="${MQ_API:-http://localhost:15672}"
DB_ROOT_PASS="${DB_ROOT_PASS:-root_secret}"
DB_PREFIXES="auth catalog inventory order payment promotion logistics procurement support notification"
TMPD="${TMPDIR:-/tmp}"
JSON='Content-Type: application/json'
FAILED=0
WARNED=0

# --- tampilan ---
if [ -t 1 ]; then R=$'\033[31m'; G=$'\033[32m'; Y=$'\033[33m'; B=$'\033[36m'; BD=$'\033[1m'; N=$'\033[0m'; else R=""; G=""; Y=""; B=""; BD=""; N=""; fi
title(){ echo; echo "${BD}${B}================================================================${N}"; echo "${BD}${B} $*${N}"; echo "${BD}${B}================================================================${N}"; }
banner(){ echo "Host: $(hostname)   Waktu: $(date '+%Y-%m-%d %H:%M:%S')   Folder: $ROOT"; }
step(){ echo; echo "${BD}>> $*${N}"; }
info(){ echo "   $*"; }
ok(){ echo "  ${G}[PASS]${N} $*"; }
bad(){ echo "  ${R}[FAIL]${N} $*"; FAILED=1; }
warn(){ echo "  ${Y}[PERHATIAN]${N} $*"; WARNED=1; }
check(){ if [ "$2" = "$3" ]; then ok "$1: $2"; else bad "$1: dapat '$2', seharusnya '$3'"; fi; }
need(){ command -v "$1" >/dev/null 2>&1 || { echo "${R}[ERROR]${N} perintah '$1' tidak ditemukan. $2"; exit 1; }; }

PY="$(command -v python3 || command -v python || true)"
[ -n "$PY" ] || { echo "[ERROR] python3 diperlukan (untuk membaca JSON)."; exit 1; }
need docker "Pasang dan jalankan Docker Desktop."
need curl "Pasang curl."

# --- alat bantu ---
jget(){ "$PY" -c "
import sys, json
try:
    v = json.loads(sys.argv[1])
    for k in sys.argv[2].split('.'):
        v = v[k]
    print(v)
except Exception:
    print('')
" "$1" "$2"; }

# Mode database terdeteksi otomatis:
#   shared      = docker-compose.yml bawaan (1 container nusantara-mysql berisi 9 database)
#   per-service = docker-compose.db-per-service.yml (nusantara-<service>-db, satu container per service)
if [ "$(docker inspect -f '{{.State.Running}}' "$MYSQL_CONTAINER" 2>/dev/null)" = "true" ]; then DB_MODE="shared"
elif docker inspect nusantara-auth-db >/dev/null 2>&1; then DB_MODE="per-service"
else DB_MODE="shared"; fi
# Compose file yang sedang dipakai stack ini (docker-compose.yml / db-per-service / microservices) terdeteksi otomatis
if [ -z "${COMPOSE_FILE:-}" ]; then
  cf="$(docker inspect -f '{{index .Config.Labels "com.docker.compose.project.config_files"}}' order-service 2>/dev/null)"
  cf="${cf%%,*}"; cf="${cf##*[\\/]}"
  if [ -n "$cf" ] && [ -f "$cf" ]; then export COMPOSE_FILE="$cf"; fi
fi
dbc(){ echo "nusantara-$1-db"; }     # nama container database milik service $1 (mode per-service)
sqlc(){ docker exec "$1" mysql -uroot -p"$DB_ROOT_PASS" -N -B -e "$2" 2>/dev/null; }   # kueri ke container tertentu
sql(){   # kueri otomatis diarahkan ke container yang benar (berdasarkan awalan <nama>_db.<tabel>)
  local q="$1" c="$MYSQL_CONTAINER" db
  if [ "$DB_MODE" = "per-service" ]; then
    db="$(echo "$q" | grep -oE '[a-z]+_db\.' | head -1 | tr -d '.')"
    [ -n "$db" ] && c="nusantara-${db%_db}-db"
  fi
  sqlc "$c" "$q"
}
pe(){ local n; n="$(sql "SELECT COUNT(*) FROM $1.processed_events")"; echo "${n:-0}"; }   # jumlah event yang sudah diproses (ledger idempotensi)
unpub(){ local n; n="$(sql "SELECT COUNT(*) FROM order_db.outbox_events WHERE published_at IS NULL")"; echo "${n:-0}"; }
rmq(){ docker exec "$MQ_CONTAINER" rabbitmqctl -q "$@" 2>/dev/null; }
qready(){ local n; n="$(rmq list_queues -p "$MQ_VHOST" name messages_ready | awk -v q="$1" '$1==q{print $2}')"; echo "${n:-0}"; }
qconsumers(){ local n; n="$(rmq list_queues -p "$MQ_VHOST" name consumers | awk -v q="$1" '$1==q{print $2}')"; echo "${n:-0}"; }
qall(){ rmq list_queues -p "$MQ_VHOST" name messages_ready messages_unacknowledged consumers | sed 's/^/   /'; }
is_running(){ [ "$(docker inspect -f '{{.State.Running}}' "$1" 2>/dev/null)" = "true" ]; }
ensure_running(){ local c; for c in "$@"; do is_running "$c" || { warn "memulihkan container $c (pengaman)"; docker start "$c" >/dev/null 2>&1; }; done; }
http_code(){ curl -s -o /dev/null -m "${CURL_TIMEOUT:-10}" -w '%{http_code}' "$@"; }
is2xx(){ case "$1" in 2??) return 0;; *) return 1;; esac; }

wait_for(){   # wait_for "<deskripsi>" <batas detik> <nama fungsi yang return 0 bila sudah terpenuhi>
  local desc="$1" timeout="$2" fn="$3" t=0
  printf "   menunggu: %s " "$desc"
  while ! "$fn"; do
    t=$((t+2)); if [ "$t" -ge "$timeout" ]; then echo " (waktu habis ${timeout}s)"; return 1; fi
    printf "."; sleep 2
  done
  echo " (${t}s)"; return 0
}

TOKEN=""
login(){
  local resp
  resp="$(curl -s -m 15 -X POST "$BASE/auth/login" -H "$JSON" -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")"
  TOKEN="$(jget "$resp" data.token)"
  [ -n "$TOKEN" ]
}
add_to_cart(){   # mencetak kode HTTP. Jalur SINKRON: order-service -> catalog-service (gRPC)
  curl -s -o "$TMPD/_cart_resp.json" -m "${CURL_TIMEOUT:-10}" -w '%{http_code}' -X POST "$BASE/orders/cart" \
    -H "Authorization: Bearer $TOKEN" -H "$JSON" -d "{\"product_id\":\"$PRODUCT_ID\",\"quantity\":1}"
}
do_checkout(){   # mencetak kode HTTP; isi balasan di $TMPD/_checkout_resp.json. Jalur ASINKRON: terbit event OrderPlaced
  curl -s -o "$TMPD/_checkout_resp.json" -m 30 -w '%{http_code}' -X POST "$BASE/orders/checkout" \
    -H "Authorization: Bearer $TOKEN" -H "$JSON" \
    -d "{\"shipping_address_id\":\"$ADDRESS_ID\",\"warehouse_id\":\"$WAREHOUSE_ID\"}"
}
run_checkouts(){  # run_checkouts N  -> mengisi OK_COUNT dan FAIL_COUNT (jeda 1,1 dtk: nomor order dibuat dari waktu)
  local n="$1" i code
  OK_COUNT=0; FAIL_COUNT=0
  for i in $(seq 1 "$n"); do
    add_to_cart >/dev/null
    code="$(do_checkout)"
    if [ "$code" = "201" ]; then OK_COUNT=$((OK_COUNT+1)); else FAIL_COUNT=$((FAIL_COUNT+1)); echo; info "checkout ke-$i gagal HTTP $code: $(head -c 160 "$TMPD/_checkout_resp.json" 2>/dev/null)"; fi
    printf "\r   [%s/%s] checkout berhasil=%s gagal=%s   " "$i" "$n" "$OK_COUNT" "$FAIL_COUNT"
    sleep "${SLEEP:-1.1}"
  done
  echo
}
require_stack(){
  step "Pemeriksaan awal: sistem harus sudah hidup"
  local c; c="$(http_code "$BASE/health")"
  if ! is2xx "$c"; then bad "gateway tidak sehat (HTTP $c). Jalankan: docker compose up --build -d, tunggu 30-60 detik, lalu ulangi."; exit 1; fi
  ok "gateway hidup (HTTP $c)"
  if ! login; then bad "login gagal (service belum siap?). Tunggu 30-60 detik lalu ulangi. Cek: docker compose ps"; exit 1; fi
  ok "login berhasil sebagai $EMAIL"
}
finish(){
  echo
  if [ "$FAILED" = "0" ]; then
    echo "${G}${BD}SELESAI: SEMUA PEMERIKSAAN LULUS${N}"
    [ "$WARNED" = "1" ] && echo "${Y}(ada catatan [PERHATIAN] di atas, baca dan jelaskan bila ditanya)${N}"
    echo "Ambil screenshot layar ini (pastikan nama host dan waktu di baris paling atas ikut terlihat)."
  else
    echo "${R}${BD}SELESAI: ADA PEMERIKSAAN GAGAL${N} - lihat pesan [FAIL] di atas."
    echo "Bantuan: docker compose ps ; docker compose logs --tail=50 <nama-service>"
    exit 1
  fi
}
# ====================== AKHIR BAGIAN BERSAMA ======================
title "E - PEMBUKTIAN ISOLASI DATA (database-per-service)"
banner
[ "$DB_MODE" = "shared" ] && { r="$(sql 'SELECT 1')"; [ "$r" = "1" ] || { bad "MySQL tidak menjawab (container $MYSQL_CONTAINER hidup?)"; exit 1; }; }
if [ "$DB_MODE" = "per-service" ]; then
  info "Mode database: per-service (satu CONTAINER MySQL per service)"
  step "1) Satu container database per service (masing-masing hanya berisi database miliknya)"
  n=0
  for p in $DB_PREFIXES; do
    c="$(dbc "$p")"
    if is_running "$c"; then
      dbs="$(sqlc "$c" "SHOW DATABASES" | grep '_db$' | tr '\n' ' ')"
      printf "   %-26s database: %s\n" "$c" "$dbs"
      [ "${dbs% }" = "${p}_db" ] && n=$((n+1)) || bad "$c berisi '$dbs', seharusnya hanya ${p}_db"
    else bad "$c tidak berjalan"; fi
  done
  [ "$n" -ge 9 ] && ok "$n container database terpisah, masing-masing 1 database" || bad "hanya $n container database sesuai harapan"
  step "2) Kredensial terpisah: tiap container punya user sendiri"
  for p in $DB_PREFIXES; do u="$(sqlc "$(dbc "$p")" "SELECT user FROM mysql.user WHERE user LIKE '%\\_user'" | tr '\n' ' ')"; printf "   %-26s user: %s\n" "$(dbc "$p")" "$u"; done
  step "3) Tidak ada foreign key yang menunjuk database lain (hasil harus kosong)"
  fk=""; for p in $DB_PREFIXES; do fk="$fk$(sqlc "$(dbc "$p")" "SELECT CONCAT(TABLE_SCHEMA,'.',TABLE_NAME,' -> ',REFERENCED_TABLE_SCHEMA) FROM information_schema.KEY_COLUMN_USAGE WHERE REFERENCED_TABLE_SCHEMA IS NOT NULL AND REFERENCED_TABLE_SCHEMA <> TABLE_SCHEMA")"; done
  [ -z "$fk" ] && ok "nol foreign key lintas database (dan memang mustahil lintas container)" || { bad "ditemukan FK lintas database:"; echo "$fk" | sed 's/^/     /'; }
  step "4) Volume terpisah dan TANPA port host (isolasi jaringan Tugas 5)"
  for p in $DB_PREFIXES; do c="$(dbc "$p")"; printf "   %-26s volume: %-18s port host: %s\n" "$c" "$(docker inspect -f '{{range .Mounts}}{{if .Name}}{{.Name}}{{end}}{{end}}' "$c" 2>/dev/null)" "$(docker port "$c" 3306 2>/dev/null | head -1)"; done
  step "5) Snapshot data historis: harga saat transaksi tersimpan di order_items"
  items="$(sql "SELECT CONCAT('produk=',product_id,' qty=',quantity,' unit_price=',unit_price,' subtotal=',subtotal) FROM order_db.order_items ORDER BY id LIMIT 5")"
  [ -n "$items" ] && { echo "$items" | sed 's/^/   /'; ok "unit_price dan subtotal tersimpan per item"; } || warn "belum ada order (jalankan skrip 01-A dulu)"
  step "6) Tabel outbox/ledger idempotensi tersedia di tiap database"
  for p in $DB_PREFIXES; do t="$(sqlc "$(dbc "$p")" "SELECT GROUP_CONCAT(TABLE_NAME) FROM information_schema.TABLES WHERE TABLE_SCHEMA='${p}_db' AND TABLE_NAME IN ('processed_events','outbox_events')")"; printf "   %-18s %s\n" "${p}_db" "${t:-(tidak ada)}"; done
  finish; exit 0
fi

step "1) Database terpisah per service"
DBS="$(sql "SHOW DATABASES" | grep '_db$')"
echo "$DBS" | sed 's/^/   /'
n="$(echo "$DBS" | grep -c .)"
[ "$n" -ge 9 ] && ok "$n database logis (satu per service)" || bad "hanya $n database ditemukan (harapan >= 9)"
info "Catatan jujur: semua berada di satu server MySQL (kompromi lab). Isolasi bersifat logis + hak akses; produksi memakai instance terpisah."

step "2) Hak akses: tiap database punya user sendiri"
sql "SELECT db,GROUP_CONCAT(DISTINCT user) FROM mysql.db WHERE db LIKE '%\\_db' GROUP BY db ORDER BY db" | awk '{printf "   %-22s user: %s\n",$1,$2}' 
rows="$(sql "SELECT COUNT(DISTINCT db) FROM mysql.db WHERE db LIKE '%\\_db'")"
[ "${rows:-0}" -ge 9 ] && ok "hak akses per database terdefinisi untuk $rows database" || warn "hak akses per database terbaca $rows (cek deployments/sql/generated/00-users.sql)"

step "3) TIDAK ADA foreign key yang menunjuk database lain (hasil harus kosong)"
fk="$(sql "SELECT CONCAT(TABLE_SCHEMA,'.',TABLE_NAME,' -> ',REFERENCED_TABLE_SCHEMA,'.',REFERENCED_TABLE_NAME) FROM information_schema.KEY_COLUMN_USAGE WHERE REFERENCED_TABLE_SCHEMA IS NOT NULL AND REFERENCED_TABLE_SCHEMA <> TABLE_SCHEMA")"
if [ -z "$fk" ]; then ok "nol foreign key lintas database"; else bad "ditemukan FK lintas database:"; echo "$fk" | sed 's/^/     /'; fi

step "4) Snapshot data historis: harga saat transaksi tersimpan di order_items"
items="$(sql "SELECT CONCAT('produk=',product_id,' qty=',quantity,' unit_price=',unit_price,' subtotal=',subtotal) FROM order_db.order_items ORDER BY id LIMIT 5")"
if [ -n "$items" ]; then echo "$items" | sed 's/^/   /'; ok "unit_price dan subtotal tersimpan per item (perubahan harga di Catalog tidak mengubah order lama)"; else warn "belum ada order (jalankan skrip 01-A dulu)"; fi

step "5) Tabel pendukung event ada di database tiap service"
for db in auth_db catalog_db inventory_db order_db payment_db promotion_db logistics_db procurement_db support_db notification_db; do
  t="$(sql "SELECT GROUP_CONCAT(TABLE_NAME) FROM information_schema.TABLES WHERE TABLE_SCHEMA='$db' AND TABLE_NAME IN ('processed_events','outbox_events')")"
  printf "   %-18s %s\n" "$db" "${t:-(tidak ada)}"
done
t="$(sql "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA='order_db' AND TABLE_NAME IN ('processed_events','outbox_events')")"
check "order_db memiliki processed_events dan outbox_events" "$t" "2"

step "6) Jumlah tabel per database"
sql "SELECT TABLE_SCHEMA,COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA LIKE '%\\_db' GROUP BY TABLE_SCHEMA ORDER BY TABLE_SCHEMA" | awk '{printf "   %-22s %s tabel\n",$1,$2}'
finish
