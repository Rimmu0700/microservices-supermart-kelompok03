#!/usr/bin/env bash
# =====================================================================================
# NAMA       : 01-A-bukti-alur-transaksi-end-to-end.sh
# KATEGORI   : PEMBUKTIAN  (sistem sehat)   -> Bagian A
# FUNGSI     : Menjalankan SATU pembelian utuh: login -> add-to-cart (SINKRON gRPC ke
#              catalog) -> checkout (event ASINKRON OrderPlaced) -> faktur & pengiriman
#              otomatis -> bayar -> status PROCESSING, lalu menunjukkan jejaknya di
#              outbox, RabbitMQ, dan ledger idempotensi.
# DIJALANKAN : bash scripts/demo/01-A-bukti-alur-transaksi-end-to-end.sh
# OPSI ENV   : WAIT_MAX=40 (batas tunggu event, detik)
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
WAIT_MAX="${WAIT_MAX:-40}"
title "A - PEMBUKTIAN: ALUR TRANSAKSI BISNIS END-TO-END"
banner
cat <<TXT

  Yang akan dibuktikan:
   - Jalur SINKRON  : order-service bertanya ke catalog-service lewat gRPC (add-to-cart)
   - Jalur ASINKRON : checkout hanya menyimpan order + event; payment & logistics bereaksi
                      sendiri lewat RabbitMQ
TXT
require_stack

step "Catat kondisi awal (ledger idempotensi tiap consumer)"
P_INV0="$(pe inventory_db)"; P_PAY0="$(pe payment_db)"; P_LOG0="$(pe logistics_db)"; P_NOT0="$(pe notification_db)"
info "processed_events awal -> inventory=$P_INV0 payment=$P_PAY0 logistics=$P_LOG0 notification=$P_NOT0"

step "1) JALUR SINKRON (gRPC): tambah ke keranjang"
info "order-service -> catalog-service:50051 (GetProduct). Jika catalog mati, langkah ini gagal."
code="$(add_to_cart)"
is2xx "$code" && ok "add-to-cart berhasil (HTTP $code), produk divalidasi catalog lewat gRPC" || { bad "add-to-cart gagal (HTTP $code): $(head -c 200 "$TMPD/_cart_resp.json")"; exit 1; }

step "2) Checkout: order + event OrderPlaced disimpan dalam 1 transaksi (Transactional Outbox)"
code="$(do_checkout)"; RESP="$(cat "$TMPD/_checkout_resp.json")"
ORDER_ID="$(jget "$RESP" data.id)"; ORDER_NO="$(jget "$RESP" data.order_number)"
if [ "$code" = "201" ] && [ -n "$ORDER_ID" ]; then ok "order $ORDER_NO dibuat (id $ORDER_ID)"; else bad "checkout gagal HTTP $code: $RESP"; exit 1; fi
info "order-service langsung membalas user; sisanya berjalan di belakang layar (asinkron)."

step "3) JALUR ASINKRON: payment-service membuat faktur otomatis (event OrderPlaced)"
invoice_ready(){ [ "$(jget "$(curl -s -m 10 "$BASE/payments/invoices/order/$ORDER_ID" -H "Authorization: Bearer $TOKEN")" data.payment_status)" = "UNPAID" ]; }
wait_for "faktur UNPAID muncul" "$WAIT_MAX" invoice_ready && ok "faktur dibuat otomatis: UNPAID" || bad "faktur belum terbentuk dalam ${WAIT_MAX}s (cek: docker compose logs --tail=50 payment-service)"

step "4) JALUR ASINKRON: logistics-service membuat pengiriman otomatis (event OrderPlaced)"
ship_ready(){ [ "$(jget "$(curl -s -m 10 "$BASE/logistics/order/$ORDER_ID" -H "Authorization: Bearer $TOKEN")" data.shipping_order.current_status)" = "READY_FOR_PICKUP" ]; }
wait_for "pengiriman READY_FOR_PICKUP muncul" "$WAIT_MAX" ship_ready && ok "pengiriman dibuat otomatis: READY_FOR_PICKUP" || bad "pengiriman belum terbentuk (cek: docker compose logs --tail=50 logistics-service)"

step "5) Pembayaran: payment-service memanggil order-service (REST internal bertoken)"
curl -s -o /dev/null -m 15 -X POST "$BASE/payments/pay" -H "Authorization: Bearer $TOKEN" -H "$JSON" \
  -d "{\"order_id\":\"$ORDER_ID\",\"payment_method_id\":\"$PAYMENT_METHOD_ID\"}"
order_processing(){ [ "$(jget "$(curl -s -m 10 "$BASE/orders/$ORDER_ID" -H "Authorization: Bearer $TOKEN")" data.order.status_code)" = "PROCESSING" ]; }
wait_for "status order menjadi PROCESSING" 20 order_processing && ok "status order: PROCESSING" || bad "status order belum PROCESSING"

step "6) JEJAK di database dan RabbitMQ (bukti event benar-benar mengalir)"
row="$(sql "SELECT event_type,routing_key,IFNULL(published_at,'BELUM-TERBIT') FROM order_db.outbox_events WHERE CAST(payload AS CHAR) LIKE '%$ORDER_ID%' ORDER BY created_at LIMIT 1")"
if [ -n "$row" ]; then info "outbox_events (tipe | routing key | published_at): $row"; echo "$row" | grep -q "BELUM-TERBIT" && bad "event belum diterbitkan ke RabbitMQ" || ok "event OrderPlaced sudah diterbitkan ke RabbitMQ (published_at terisi)"; else warn "baris outbox untuk order ini tidak ditemukan lewat pencarian payload"; fi
sleep 2
P_INV1="$(pe inventory_db)"; P_PAY1="$(pe payment_db)"; P_LOG1="$(pe logistics_db)"; P_NOT1="$(pe notification_db)"
info "processed_events sekarang -> inventory=$P_INV1 (+$((P_INV1-P_INV0))) payment=$P_PAY1 (+$((P_PAY1-P_PAY0))) logistics=$P_LOG1 (+$((P_LOG1-P_LOG0))) notification=$P_NOT1 (+$((P_NOT1-P_NOT0)))"
[ "$P_PAY1" -gt "$P_PAY0" ] && [ "$P_LOG1" -gt "$P_LOG0" ] && ok "payment dan logistics mencatat event di ledger idempotensi" || warn "ledger belum bertambah di semua consumer"
echo "   Isi queue sekarang (Ready harus 0 = semua pesan sudah diproses):"; qall
finish
