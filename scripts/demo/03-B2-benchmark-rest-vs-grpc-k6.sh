#!/usr/bin/env bash
# =====================================================================================
# NAMA       : 03-B2-benchmark-rest-vs-grpc-k6.sh
# KATEGORI   : PEMBUKTIAN  (sistem sehat)   -> Bagian B (langkah B4)
# FUNGSI     : Adu kinerja REST (HTTP/1.1+JSON) vs gRPC (HTTP/2+Protobuf) memakai k6,
#              lalu mencetak TABEL PERBANDINGAN (avg, p50, p95, p99, req/s, selisih %).
#              Keluaran lengkap k6 disimpan di folder hasil-demo/ untuk screenshot/laporan.
# DIJALANKAN : bash scripts/demo/03-B2-benchmark-rest-vs-grpc-k6.sh
#              QUICK=1 bash scripts/demo/03-B2-benchmark-rest-vs-grpc-k6.sh   (uji cepat 10 VU / 5 dtk)
# OPSI ENV   : SCENARIOS="50:30s 200:60s"   (format VU:durasi, bawaan sesuai soal Tugas 2)
# LAMA       : bawaan sekitar 3 menit (30s+30s+60s+60s plus jeda)
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
need k6 "Pasang k6 dari https://k6.io lalu ulangi."
if [ -z "$(docker port catalog-service 50051 2>/dev/null | head -1)" ]; then
  echo "[PERHATIAN] port gRPC 50051 tidak dipublikasikan (isolasi Tugas 5). Benchmark gRPC dari host butuh override sementara:"
  echo "  docker compose -f docker-compose.yml -f docker-compose.grpc-debug.yml up -d catalog-service"
  exit 0
fi
SCENARIOS="${SCENARIOS:-50:30s 200:60s}"
[ "${QUICK:-0}" = "1" ] && SCENARIOS="10:5s"
STAMP="$(date +%Y%m%d-%H%M%S)"
OUT_REL="../../hasil-demo/benchmark-$STAMP"      # relatif terhadap tests/benchmark (aman untuk k6 di Windows)
OUT="hasil-demo/benchmark-$STAMP"      # relatif terhadap folder proyek (aman di Windows)
mkdir -p "$OUT"

title "B2 - BENCHMARK REST vs gRPC (k6)"
banner
require_stack
info "Skenario: $SCENARIOS   | keluaran disimpan di: hasil-demo/benchmark-$STAMP"

for sc in $SCENARIOS; do
  VUS="${sc%%:*}"; DUR="${sc##*:}"
  step "REST   - $VUS pengguna virtual, $DUR  (HTTP/1.1 + JSON)"
  ( cd tests/benchmark && k6 run -e VUS="$VUS" -e DURATION="$DUR" --summary-export="$OUT_REL/rest-$VUS.json" k6-rest.js ) 2>&1 | tee "$OUT/rest-$VUS.log"
  sleep 3
  step "gRPC   - $VUS pengguna virtual, $DUR  (HTTP/2 + Protobuf)"
  ( cd tests/benchmark && k6 run -e VUS="$VUS" -e DURATION="$DUR" --summary-export="$OUT_REL/grpc-$VUS.json" k6-grpc.js ) 2>&1 | tee "$OUT/grpc-$VUS.log"
  sleep 3
done

step "TABEL PERBANDINGAN (angka asli dari mesin ini)"
"$PY" - "$OUT" $SCENARIOS <<'PYEOF' | tee "$OUT/ringkasan.txt"
import json, sys, os
out = sys.argv[1]; scenarios = sys.argv[2:]
def val(d, k):
    d = d.get("values", d)
    return d.get(k)
def load(kind, vus):
    p = os.path.join(out, f"{kind}-{vus}.json")
    if not os.path.exists(p): return None
    m = json.load(open(p))["metrics"]
    dur = m.get("http_req_duration" if kind == "rest" else "grpc_req_duration")
    reqs = m.get("http_reqs") if kind == "rest" else (m.get("grpc_reqs") or m.get("iterations"))
    chk = m.get("checks")
    if dur is None: return None
    return {"avg": val(dur,"avg"), "p50": val(dur,"med"), "p95": val(dur,"p(95)"), "p99": val(dur,"p(99)"),
            "rps": val(reqs,"rate") if reqs else None,
            "ok": (val(chk,"value") if chk else None)}
def f(x, n=2): return "-" if x is None else f"{x:.{n}f}"
def pct_lower(r, g):   # makin kecil makin baik
    return "-" if not r or g is None else f"{(r-g)/r*100:+.1f}% (gRPC lebih {'cepat' if g<r else 'lambat'})"
def pct_higher(r, g):
    return "-" if not r or g is None else f"{(g-r)/r*100:+.1f}% (gRPC lebih {'tinggi' if g>r else 'rendah'})"
for sc in scenarios:
    vus, dur = sc.split(":")
    r, g = load("rest", vus), load("grpc", vus)
    print(f"\n=== Skenario {vus} pengguna virtual / {dur} ===")
    if not r or not g:
        print("  data tidak lengkap (lihat berkas .log di folder hasil)"); continue
    rows = [("Latensi rata-rata (ms)", f(r["avg"]), f(g["avg"]), pct_lower(r["avg"], g["avg"])),
            ("Latensi p50 (ms)",       f(r["p50"]), f(g["p50"]), pct_lower(r["p50"], g["p50"])),
            ("Latensi p95 (ms)",       f(r["p95"]), f(g["p95"]), pct_lower(r["p95"], g["p95"])),
            ("Latensi p99 (ms)",       f(r["p99"]), f(g["p99"]), pct_lower(r["p99"], g["p99"])),
            ("Throughput (req/s)",     f(r["rps"],1), f(g["rps"],1), pct_higher(r["rps"], g["rps"])),
            ("Checks lulus (0-1)",     f(r["ok"],3), f(g["ok"],3), "")]
    print(f"  {'Metrik':<26}{'REST':>12}{'gRPC':>12}   Selisih")
    for a,b,c,d in rows: print(f"  {a:<26}{b:>12}{c:>12}   {d}")
print("\nUkuran payload per respons: jalankan skrip 02-B1 (bagian 3).")
PYEOF
echo
info "Berkas hasil: hasil-demo/benchmark-$STAMP/ (rest-*.log, grpc-*.log, ringkasan.txt)"
info "Catatan jujur: pada satu mesin lokal hasil bergantung pada beban mesin; laporkan apa adanya."
finish
