#!/usr/bin/env bash
# =====================================================================================
# NAMA       : 02-B1-bukti-grpc-server-klien-dan-ukuran-payload.sh
# KATEGORI   : PEMBUKTIAN  (sistem sehat)   -> Bagian B (langkah B1-B3)
# FUNGSI     : (1) Membuktikan server gRPC catalog hidup di :50051
#              (2) Memanggil gRPC langsung: Unary GetProduct, NotFound, Batch
#              (3) Membandingkan ukuran balasan JSON (REST) vs Protobuf (gRPC)
# DIJALANKAN : bash scripts/demo/02-B1-bukti-grpc-server-klien-dan-ukuran-payload.sh
# PRASYARAT  : Go (untuk klien). Tanpa Go, dipakai jalur cadangan k6 (1 pengguna, 3 detik).
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
title "B1 - PEMBUKTIAN gRPC: server aktif, klien Unary + Batch, ukuran payload"
banner
require_stack

step "1) Server gRPC catalog-service hidup di port 50051?"
line="$(docker compose logs catalog-service 2>/dev/null | grep -i "grpc server listening" | tail -1)"
[ -n "$line" ] && ok "log server: ${line##*|}" || bad "log 'gRPC server listening' tidak ditemukan (docker compose logs catalog-service)"
p="$(docker port catalog-service 50051 2>/dev/null | head -1)"
[ -n "$p" ] && ok "port 50051 terpublikasi ke laptop ($p). Port 3000 = REST, 50051 = gRPC: satu service, dua pintu" || warn "port 50051 tidak dipublikasikan ke host (isolasi Tugas 5)"

if [ -z "$p" ]; then
  warn "klien gRPC dari host butuh port 50051. Aktifkan sementara (hanya localhost):"
  info "docker compose -f docker-compose.microservices.yml -f docker-compose.grpc-debug.yml up -d catalog-service"
  info "lalu ulangi skrip ini. Setelah selesai: docker compose -f docker-compose.microservices.yml up -d catalog-service"
  finish; exit 0
fi

step "2) Panggil gRPC langsung (klien bawaan repo: cmd/grpcclient)"
if command -v go >/dev/null 2>&1; then
  GO_DIR="backend"; [ -d tools/grpc-tools ] && GO_DIR="tools/grpc-tools"
  out="$(cd "$GO_DIR" && go run ./cmd/grpcclient 2>&1)"; rc=$?
  echo "$out" | sed 's/^/   /'
  [ $rc -eq 0 ] || bad "klien gRPC keluar dengan kode $rc"
  echo "$out" | grep -q "\[Unary\] p0000001" && ok "Unary GetProduct berhasil (1 produk)" || bad "Unary GetProduct tidak menghasilkan produk"
  echo "$out" | grep -q "NotFound sesuai harapan" && ok "produk tidak ada -> kode NotFound yang rapi (bukan crash)" || bad "kasus NotFound tidak sesuai"
  echo "$out" | grep -q "\[Batch\]" && ok "BatchGetProducts berhasil (banyak produk dalam 1 panggilan; yang tidak ada dilaporkan)" || bad "Batch gagal"
elif command -v k6 >/dev/null 2>&1; then
  warn "Go tidak ada, memakai jalur cadangan: k6 gRPC 1 pengguna selama 3 detik"
  (cd tests/benchmark && k6 run -e VUS=1 -e DURATION=3s k6-grpc.js 2>&1 | tail -25 | sed 's/^/   /')
else
  bad "Go dan k6 sama-sama tidak ada. Pasang salah satunya."
fi

step "3) Ukuran balasan: JSON (REST) vs Protobuf (gRPC) untuk produk yang sama"
JSON_BYTES="$(curl -s -m 10 "$BASE/catalog/products/$PRODUCT_ID/summary" | wc -c | tr -d ' ')"
info "REST/JSON : ${JSON_BYTES} bytes   ($BASE/catalog/products/<id>/summary)"
if command -v go >/dev/null 2>&1; then
  GO_DIR="backend"; [ -d tools/grpc-tools ] && GO_DIR="tools/grpc-tools"
  pout="$(cd "$GO_DIR" && go run ./cmd/payloadsize 2>&1)"
  PB_BYTES="$(echo "$pout" | grep -o '[0-9][0-9]* bytes' | head -1 | awk '{print $1}')"
  info "Protobuf  : ${PB_BYTES:-?} bytes   (cmd/payloadsize)"
  if [ -n "${PB_BYTES:-}" ] && [ "${JSON_BYTES:-0}" -gt 0 ]; then
    pct="$("$PY" -c "print(round((int('$JSON_BYTES')-int('$PB_BYTES'))/int('$JSON_BYTES')*100,1))")"
    ok "Protobuf lebih ringkas ${pct}% dibanding JSON (catatan: JSON REST memuat pembungkus success/data)"
  else
    warn "ukuran tidak dapat dihitung (cek keluaran payloadsize)"
  fi
else
  warn "Go tidak ada: ukuran Protobuf tidak bisa dihitung di sini"
fi
finish
