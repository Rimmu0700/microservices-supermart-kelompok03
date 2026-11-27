#!/usr/bin/env bash
# =====================================================================================
# NAMA       : 14-D5-gangguan-container-acak-pilihan-dosen.sh
# KATEGORI   : SIMULASI GANGGUAN (fault-injection)  -> Bagian D5
# FUNGSI     : Mematikan SATU container (acak atau yang kamu sebut), lalu menampilkan TABEL
#              kondisi SEBELUM / SAAT MATI / SESUDAH PULIH untuk 5 pemeriksaan, dan
#              menjelaskan dampak yang diprediksi. Cocok untuk "dosen memilih container acak".
# DIJALANKAN : bash scripts/demo/14-D5-gangguan-container-acak-pilihan-dosen.sh            (acak)
#              bash scripts/demo/14-D5-gangguan-container-acak-pilihan-dosen.sh payment-service
#              bash scripts/demo/14-D5-gangguan-container-acak-pilihan-dosen.sh --list
# OPSI ENV   : DOWN_SECONDS=8 (lama container dibiarkan mati sebelum pemeriksaan)
# PENGAMAN   : container otomatis dinyalakan kembali bila skrip dihentikan.
# CATATAN    : setiap run membuat beberapa order uji (add-to-cart + checkout) sebagai pemeriksaan.
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
if [ "$DB_MODE" = "shared" ]; then DBLIST="nusantara-mysql"; else DBLIST=""; for p in $DB_PREFIXES; do DBLIST="$DBLIST $(dbc "$p")"; done; fi
CANDIDATES="auth-service catalog-service inventory-service order-service payment-service promotion-service logistics-service procurement-service support-service notification-worker nusantara-api-gateway nusantara-frontend nusantara-rabbitmq $DBLIST"
if [ "${1:-}" = "--list" ]; then echo "Container yang bisa dipilih:"; for c in $CANDIDATES; do echo "  $c"; done; exit 0; fi
arr=($CANDIDATES)
if [ -z "${1:-}" ] || [ "${1:-}" = "acak" ]; then TARGET="${arr[$((RANDOM % ${#arr[@]}))]}"; else TARGET="$1"; fi
case " $CANDIDATES " in *" $TARGET "*) ;; *) echo "[ERROR] container '$TARGET' tidak dikenal. Lihat: bash $0 --list"; exit 1;; esac
DOWN_SECONDS="${DOWN_SECONDS:-8}"
trap 'ensure_running "$TARGET"' EXIT

probe(){   # mencetak 5 kode HTTP: health login katalog add-to-cart checkout
  local h l c a k
  h="$(http_code "$BASE/health")"
  l="$(curl -s -o /dev/null -m 8 -w '%{http_code}' -X POST "$BASE/auth/login" -H "$JSON" -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")"
  c="$(http_code -H "Authorization: Bearer $TOKEN" "$BASE/catalog/products/$PRODUCT_ID")"
  a="$(add_to_cart)"; k="$(do_checkout)"
  echo "$h $l $c $a $k"
}
queue_sum(){ if is_running "$MQ_CONTAINER"; then rmq list_queues -p "$MQ_VHOST" name messages_ready | awk '$1!="supermart.dlq"{s+=$2} END{print s+0}'; else echo "broker-mati"; fi; }
show_row(){ read -r h l c a k <<<"$2"; printf "   %-27s %-8s %-8s %-9s %-13s %-9s %-10s\n" "$1" "$h" "$l" "$c" "$a" "$k" "$3"; }

title "D5 - SIMULASI GANGGUAN: container pilihan dosen = $TARGET"
banner
require_stack
echo
echo "   Arti kode: 2xx = OK | 000 = tidak terhubung | 502/503/504 = gateway tidak bisa ke service | lainnya = galat aplikasi"
echo "   Kolom 'add-to-cart' = jalur SINKRON gRPC ke catalog; 'checkout' = jalur ASINKRON (event)."

step "1) Pemeriksaan SEBELUM gangguan"
printf "   %-27s %-8s %-8s %-9s %-13s %-9s %-10s\n" "KONDISI" "health" "login" "katalog" "add-to-cart" "checkout" "antrean"
BEFORE="$(probe)"; show_row "sebelum" "$BEFORE" "$(queue_sum)"

step "2) [GANGGUAN] Mematikan $TARGET"
docker stop "$TARGET" >/dev/null 2>&1 && ok "$TARGET dimatikan" || bad "gagal mematikan $TARGET"
sleep "$DOWN_SECONDS"
TOKEN_SAVED="$TOKEN"   # token JWT lama tetap dipakai (stateless) walau auth-service mati
printf "   %-27s %-8s %-8s %-9s %-13s %-9s %-10s\n" "KONDISI" "health" "login" "katalog" "add-to-cart" "checkout" "antrean"
DURING="$(probe)"; show_row "saat $TARGET mati" "$DURING" "$(queue_sum)"

step "3) Dampak yang diprediksi untuk $TARGET (cocokkan dengan tabel di atas)"
case "$TARGET" in
  inventory-service)    info "Checkout TETAP sukses; pesan reservasi menunggu di inventory.stock-reservation.q (asinkron). Pembatalan order (release stok via REST) akan gagal.";;
  payment-service)      info "Faktur tidak terbentuk dan endpoint bayar gagal (502); order/inventory/logistics tetap jalan; OrderPlaced menunggu di queue payment.";;
  logistics-service)    info "Shipping order tertunda dan status kirim tidak bisa diubah; order dan pembayaran tetap jalan.";;
  notification-worker)  info "Notifikasi tertunda; seluruh transaksi tetap jalan (consumer murni). Pesan tertahan terkirim setelah hidup.";;
  catalog-service)      info "add-to-cart GAGAL (gRPC timeout 2 detik) dan browsing katalog gagal. Itu konsekuensi jalur SINKRON; checkout keranjang yang sudah ada bisa jalan karena harga tersimpan.";;
  order-service)        info "Cart dan checkout GAGAL; kompensasi StockReservationFailed menunggu sampai order-service hidup. Payment/inventory/catalog tetap hidup.";;
  auth-service)         info "Login BARU gagal; token JWT lama tetap valid karena diverifikasi lokal dengan JWT_SECRET bersama (stateless).";;
  nusantara-rabbitmq)   info "Event baru tidak bisa terbit sementara, TETAPI checkout tetap sukses: event aman di outbox_events lalu diterbitkan ulang saat broker kembali.";;
  nusantara-mysql)      info "Semua service yang butuh database gagal (checkout, login, dst). Pesan yang sudah ada di broker aman. Service mungkin butuh waktu/restart untuk menyambung ulang.";;
  nusantara-api-gateway) info "API Gateway (KrakenD) mati: seluruh API publik tidak terjangkau (kode 000 di :8000, 502 lewat frontend), tetapi komunikasi internal gRPC/event antar-service tetap berjalan. Gateway stateless sehingga pulih seketika dengan docker start.";;
  nusantara-frontend)   info "UI hilang; backend dan API tetap utuh (lihat kode HTTP API normal).";;
  nusantara-*-db)       info "Database milik SATU service mati: hanya service pemiliknya yang terdampak (isolasi data); service lain tetap berjalan. Pemilik gagal melayani request yang butuh DB sampai container DB hidup.";;
  *)                    info "Service pendukung; transaksi utama checkout seharusnya tetap berjalan (periksa tabel).";;
esac

step "4) [PEMULIHAN] Menyalakan kembali $TARGET"
docker start "$TARGET" >/dev/null 2>&1 && ok "$TARGET dinyalakan"
TOKEN="$TOKEN_SAVED"
recovered(){ login >/dev/null 2>&1; is2xx "$(http_code "$BASE/health")" && login && is2xx "$(add_to_cart)"; }
wait_for "sistem pulih (health, login, add-to-cart OK)" 150 recovered && ok "sistem pulih sepenuhnya" || warn "belum pulih penuh dalam 150s (cek: docker compose ps ; docker compose logs --tail=50 $TARGET)"

step "5) Pemeriksaan SESUDAH pemulihan"
printf "   %-27s %-8s %-8s %-9s %-13s %-9s %-10s\n" "KONDISI" "health" "login" "katalog" "add-to-cart" "checkout" "antrean"
AFTER="$(probe)"; show_row "sesudah pulih" "$AFTER" "$(queue_sum)"
read -r _ _ _ a k <<<"$AFTER"
is2xx "$a" && ok "add-to-cart (gRPC) normal kembali" || warn "add-to-cart belum normal (HTTP $a)"
[ "$k" = "201" ] && ok "checkout normal kembali" || warn "checkout belum normal (HTTP $k)"
echo
echo "   Ringkasan tabel: ${BD}SEBELUM${N} -> $BEFORE | ${BD}SAAT MATI${N} -> $DURING | ${BD}SESUDAH${N} -> $AFTER"
finish
