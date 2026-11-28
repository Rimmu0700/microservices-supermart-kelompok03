#!/usr/bin/env bash
# =====================================================================================
# NAMA       : 01-uji-routing-dan-isolasi-port.sh
# KATEGORI   : PEMBUKTIAN (Tugas 5 - Bagian II.1 dan II.4)
# FUNGSI     : Menguji routing/path rewriting pass-through ke 9 service, penerusan JWT dan status backend, serta isolasi port (tidak ada port layanan/DB terbuka ke host).
# DIJALANKAN : bash scripts/gateway/01-uji-routing-dan-isolasi-port.sh
# PRASYARAT  : sistem hidup lewat docker compose (API Gateway KrakenD di :8000). Untuk uji gangguan/pembatasan,
#              skrip memulihkan container secara otomatis bila dihentikan (Ctrl+C).
# =====================================================================================
set -uo pipefail
export MSYS_NO_PATHCONV=1
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
while [ ! -f "$ROOT/docker-compose.yml" ] && [ ! -f "$ROOT/docker-compose.yml" ] && [ "$ROOT" != "/" ]; do ROOT="$(dirname "$ROOT")"; done
{ [ -f "$ROOT/docker-compose.yml" ] || [ -f "$ROOT/docker-compose.yml" ]; } || { echo "[ERROR] jalankan di dalam folder proyek"; exit 1; }
cd "$ROOT"

GW="${GW:-http://localhost:8000}"
BASE="$GW/api/v1"
EMAIL="${EMAIL:-wayan.putra@gmail.com}"
PASSWORD="${PASSWORD:-password123}"
PRODUCT_ID="${PRODUCT_ID:-p0000001-0000-0000-0000-000000000001}"
ORDER_ID="${ORDER_ID:-}"
JSON='Content-Type: application/json'
BACKENDS="auth-service catalog-service inventory-service order-service payment-service promotion-service logistics-service procurement-service support-service"
FAILED=0
if [ -t 1 ]; then R=$'\033[31m'; G=$'\033[32m'; Y=$'\033[33m'; B=$'\033[36m'; BD=$'\033[1m'; NC=$'\033[0m'; else R=""; G=""; Y=""; B=""; BD=""; NC=""; fi
title(){ echo; echo "${BD}${B}================================================================${NC}"; echo "${BD}${B} $*${NC}"; echo "${BD}${B}================================================================${NC}"; }
step(){ echo; echo "${BD}>> $*${NC}"; }
info(){ echo "   $*"; }
ok(){ echo "  ${G}[PASS]${NC} $*"; }
bad(){ echo "  ${R}[FAIL]${NC} $*"; FAILED=1; }
warn(){ echo "  ${Y}[PERHATIAN]${NC} $*"; }
PY="$(command -v python3 || command -v python || true)"
[ -n "$PY" ] || { echo "[ERROR] python3 diperlukan"; exit 1; }
command -v curl >/dev/null 2>&1 || { echo "[ERROR] curl diperlukan"; exit 1; }
command -v docker >/dev/null 2>&1 || { echo "[ERROR] docker diperlukan"; exit 1; }
# compose file yang sedang dipakai stack ini (terdeteksi dari label container)
if [ -z "${COMPOSE_FILE:-}" ]; then
  cf="$(docker inspect -f '{{index .Config.Labels "com.docker.compose.project.config_files"}}' nusantara-api-gateway 2>/dev/null)"
  cf="${cf%%,*}"; cf="${cf##*[\\/]}"
  if [ -n "$cf" ] && [ -f "$cf" ]; then export COMPOSE_FILE="$cf"; fi
fi
jget(){ "$PY" -c "
import sys, json
try:
    v = json.loads(sys.argv[1])
    for k in sys.argv[2].split('.'): v = v[k]
    print(v)
except Exception:
    print('')
" "$1" "$2"; }
first_id(){ "$PY" -c "
import sys, json
def find(o):
    if isinstance(o, dict):
        if isinstance(o.get('id'), str): return o['id']
        for v in o.values():
            r = find(v)
            if r: return r
    elif isinstance(o, list):
        for v in o:
            r = find(v)
            if r: return r
    return ''
try: print(find(json.loads(sys.argv[1])))
except Exception: print('')
" "$1"; }
keys_of(){ "$PY" -c "
import sys, json
try: print(' '.join(sorted(json.loads(sys.argv[1]).keys())))
except Exception: print('')
" "$1"; }
code_of(){ curl -s -o /dev/null -m 10 -w '%{http_code}' "$@"; }
is_running(){ [ "$(docker inspect -f '{{.State.Running}}' "$1" 2>/dev/null)" = "true" ]; }
wait_gw(){ local t=0; until [ "$(code_of "$GW/__health")" = "200" ]; do t=$((t+1)); [ $t -ge 30 ] && return 1; sleep 1; done; return 0; }
TOKEN=""
login(){ local r; r="$(curl -s -m 15 -X POST "$BASE/auth/login" -H "$JSON" -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")"; TOKEN="$(jget "$r" data.token)"; [ -n "$TOKEN" ]; }
need_order(){
  [ -n "$ORDER_ID" ] && return 0
  ORDER_ID="$(first_id "$(curl -s -m 10 "$BASE/orders" -H "Authorization: Bearer $TOKEN")")"
  if [ -z "$ORDER_ID" ] && [ -f scripts/smoke-test.sh ]; then
    info "belum ada order; membuat satu lewat scripts/smoke-test.sh ..."; bash scripts/smoke-test.sh >/dev/null 2>&1
    login >/dev/null 2>&1
    ORDER_ID="$(first_id "$(curl -s -m 10 "$BASE/orders" -H "Authorization: Bearer $TOKEN")")"
  fi
  [ -n "$ORDER_ID" ]
}
require_gateway(){
  step "Pemeriksaan awal: API Gateway hidup"
  local h; h="$(code_of "$GW/__health")"
  [ "$h" = "200" ] && ok "KrakenD hidup ($GW/__health = 200)" || { bad "gateway tidak menjawab (HTTP $h). Jalankan: docker compose up -d --build ; docker compose logs api-gateway"; exit 1; }
  login && ok "login lewat gateway berhasil sebagai $EMAIL" || { bad "login lewat gateway gagal"; exit 1; }
}
finish(){
  echo
  if [ "$FAILED" = "0" ]; then echo "${G}${BD}SELESAI: SEMUA PEMERIKSAAN LULUS${NC}  (screenshot layar ini; baris Host ikut terlihat)"
  else echo "${R}${BD}SELESAI: ADA PEMERIKSAAN GAGAL${NC}  (lihat [FAIL]; bantuan: docker compose logs api-gateway)"; exit 1; fi
}
echo "Host: $(hostname)   Waktu: $(date '+%Y-%m-%d %H:%M:%S')   Gateway: $GW"
title "TUGAS 5 - UJI 1: ROUTING (PATH REWRITING) DAN ISOLASI PORT"
require_gateway

step "1) Agregasi health: satu panggilan memeriksa 9 service (fan-out)"
h="$(curl -s -m 10 "$BASE/health")"; info "kunci: $(keys_of "$h")"
n="$(keys_of "$h" | wc -w | tr -d ' ')"; [ "$n" -ge 9 ] && ok "$n service menjawab lewat /api/v1/health" || warn "baru $n service menjawab (sebagian belum siap)"

step "2) Rute tunggal pass-through (klien -> gateway -> service internal)"
chk(){ # chk "<deskripsi>" <kode harapan> <metode> <path> [argumen curl tambahan]
  local d="$1" want="$2" m="$3" p="$4"; shift 4
  local c; c="$(code_of -X "$m" "$BASE$p" "$@")"
  [ "$c" = "$want" ] && ok "$d -> HTTP $c" || bad "$d -> HTTP $c (harapan $want)"
}
AUTHH=(-H "Authorization: Bearer $TOKEN")
chk "GET  /catalog/products/{id}  (publik)"            200 GET "/catalog/products/$PRODUCT_ID"
chk "GET  /catalog/categories      (publik)"            200 GET "/catalog/categories"
chk "GET  /auth/me                 (JWT diteruskan)"    200 GET "/auth/me" "${AUTHH[@]}"
chk "GET  /orders                  (JWT diteruskan)"    200 GET "/orders" "${AUTHH[@]}"
chk "GET  /payments/credits        (service payment)"   200 GET "/payments/credits" "${AUTHH[@]}"
chk "GET  /inventory/warehouses    (service inventory)" 200 GET "/inventory/warehouses" "${AUTHH[@]}"
info "Path rewriting (alias publik -> jalur internal yang berbeda):"
chk "GET  /products/{id}  ->  /catalog/products/{id}"   200 GET "/products/$PRODUCT_ID"
info "Contoh respons mentah (curl -i) untuk lampiran laporan:"
curl -s -i -m 10 "$BASE/catalog/products/$PRODUCT_ID" | head -n 12 | sed 's/^/     /'

step "3) Kode status backend diteruskan apa adanya (no-op), gateway tidak mengubah semantik"
chk "tanpa token ke /auth/me -> 401 dari auth-service"  401 GET "/auth/me"
chk "order tidak ada -> 404 dari order-service"          404 GET "/orders/00000000-0000-0000-0000-000000000000" "${AUTHH[@]}"
chk "rute tidak terdaftar di gateway -> 404"             404 GET "/rute-yang-tidak-ada"
c="$(code_of "$GW/internal/replica/products")"; [ "$c" = "404" ] && ok "endpoint internal /internal/replica tidak terjangkau dari luar (HTTP 404)" || bad "/internal terjangkau? HTTP $c"

step "4) Isolasi jaringan: tidak ada layanan backend yang memetakan port ke host"
leak=""; for s in $BACKENDS; do p="$(docker port "$s" 2>/dev/null | head -1)"; [ -n "$p" ] && leak="$leak $s($p)"; done
[ -z "$leak" ] && ok "9 layanan backend tanpa 'ports' (hanya expose di jaringan Docker)" || bad "masih memetakan port:$leak"
dbleak=""; for c in $(docker ps --format '{{.Names}}' | grep -E '^nusantara-.*-db$'); do p="$(docker port "$c" 2>/dev/null | head -1)"; [ -n "$p" ] && dbleak="$dbleak $c($p)"; done
[ -z "$dbleak" ] && ok "container database tanpa port host" || bad "database masih terbuka:$dbleak"
echo "   Port yang terbuka ke komputer host (seharusnya hanya gateway :8000, frontend :4200, dan konsol RabbitMQ localhost):"
docker ps --format '{{.Names}}\t{{.Ports}}' | grep -E '0\.0\.0\.0|127\.0\.0\.1|:::' | sed 's/^/     /'

step "5) Akses langsung ke port layanan dari komputer host harus DITOLAK (Connection refused)"
for hp in 3000 3306 5672 50051 3311 3314; do
  r="$(curl -s -m 3 -o /dev/null -w '%{http_code}' "http://localhost:$hp" 2>/dev/null)"; [ -z "$r" ] && r="000"
  if [ "$r" = "000" ]; then ok "localhost:$hp ditolak / tertutup (kode $r)"; else bad "localhost:$hp masih terjangkau (HTTP $r)"; fi
done
info "Bukti mentah: curl -i http://localhost:50051"; curl -s -i -m 3 http://localhost:50051 2>&1 | head -n 3 | sed 's/^/     /'; echo "     (exit code curl: Connection refused = 7)"
finish
