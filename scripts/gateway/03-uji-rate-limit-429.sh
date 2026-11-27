#!/usr/bin/env bash
# =====================================================================================
# NAMA       : 03-uji-rate-limit-429.sh
# KATEGORI   : PEMBUKTIAN (Tugas 5 - Bagian III)
# FUNGSI     : Membanjiri rute kritis login hingga melewati kuota, membuktikan HTTP 429, header mentah, dan pemulihan token bucket.
# DIJALANKAN : bash scripts/gateway/03-uji-rate-limit-429.sh
# PRASYARAT  : sistem hidup lewat docker compose (API Gateway KrakenD di :8000). Untuk uji gangguan/pembatasan,
#              skrip memulihkan container secara otomatis bila dihentikan (Ctrl+C).
# =====================================================================================
set -uo pipefail
export MSYS_NO_PATHCONV=1
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
while [ ! -f "$ROOT/docker-compose.yml" ] && [ ! -f "$ROOT/docker-compose.microservices.yml" ] && [ "$ROOT" != "/" ]; do ROOT="$(dirname "$ROOT")"; done
{ [ -f "$ROOT/docker-compose.yml" ] || [ -f "$ROOT/docker-compose.microservices.yml" ]; } || { echo "[ERROR] jalankan di dalam folder proyek"; exit 1; }
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
if [ -t 1 ]; then R=$'\033[31m'; G=$'\033[32m'; Y=$'\033[33m'; B=$'\033[36m'; BD=$'\033[1m'; N=$'\033[0m'; else R=""; G=""; Y=""; B=""; BD=""; N=""; fi
title(){ echo; echo "${BD}${B}================================================================${N}"; echo "${BD}${B} $*${N}"; echo "${BD}${B}================================================================${N}"; }
step(){ echo; echo "${BD}>> $*${N}"; }
info(){ echo "   $*"; }
ok(){ echo "  ${G}[PASS]${N} $*"; }
bad(){ echo "  ${R}[FAIL]${N} $*"; FAILED=1; }
warn(){ echo "  ${Y}[PERHATIAN]${N} $*"; }
PY="$(command -v python3 || command -v python || true)"
[ -n "$PY" ] || { echo "[ERROR] python3 diperlukan"; exit 1; }
command -v curl >/dev/null 2>&1 || { echo "[ERROR] curl diperlukan"; exit 1; }
command -v docker >/dev/null 2>&1 || { echo "[ERROR] docker diperlukan"; exit 1; }
# compose file yang sedang dipakai stack ini (terdeteksi dari label container)
if [ -z "${COMPOSE_FILE:-}" ]; then
  cf="$(docker inspect -f '{{index .Config.Labels "com.docker.compose.project.config_files"}}' api-gateway 2>/dev/null)"
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
  if [ "$FAILED" = "0" ]; then echo "${G}${BD}SELESAI: SEMUA PEMERIKSAAN LULUS${N}  (screenshot layar ini; baris Host ikut terlihat)"
  else echo "${R}${BD}SELESAI: ADA PEMERIKSAAN GAGAL${N}  (lihat [FAIL]; bantuan: docker compose logs api-gateway)"; exit 1; fi
}
echo "Host: $(hostname)   Waktu: $(date '+%Y-%m-%d %H:%M:%S')   Gateway: $GW"
N="${N:-15}"
title "TUGAS 5 - UJI 3: RATE LIMITING (HTTP 429) PADA RUTE KRITIS POST /api/v1/auth/login"
cat <<TXT

  Kebijakan (gateway/routes/identity.json): client_max_rate=10 per menit, client_capacity=10, strategy=ip
  Algoritma: TOKEN BUCKET. Ember berisi 10 token; tiap permintaan memakai 1 token; token terisi kembali 1 tiap 6 detik.
  Kunci klien: alamat IP. (Risiko NAT: banyak pengguna di balik satu IP berbagi satu ember yang sama.)
TXT
step "0) Reset hitungan limiter dengan restart api-gateway"
docker compose restart api-gateway >/dev/null 2>&1 && ok "api-gateway di-restart" || warn "gagal restart; hitungan mungkin tidak bersih"
wait_gw && ok "gateway siap" || { bad "gateway tidak siap"; exit 1; }

step "1) Banjir permintaan: $N login beruntun dalam waktu singkat"
codes=""; for i in $(seq 1 "$N"); do
  c="$(curl -s -o /dev/null -m 10 -w '%{http_code}' -X POST "$BASE/auth/login" -H "$JSON" -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")"
  codes="$codes $c"; echo "     Req #$i -> HTTP $c"
done
ok_n=$(echo "$codes" | tr ' ' '\n' | grep -c '^200$'); lim_n=$(echo "$codes" | tr ' ' '\n' | grep -c '^429$')
info "ringkasan: diterima (200) = $ok_n | ditolak (429) = $lim_n"
[ "$ok_n" -eq 10 ] && ok "tepat 10 permintaan pertama diterima (kapasitas ember = 10)" || warn "diterima $ok_n (harapan 10; token terisi ulang selama pengujian bisa menambah 1-2)"
[ "$lim_n" -ge 1 ] && ok "permintaan berlebih ditolak dengan HTTP 429 Too Many Requests" || bad "tidak ada respons 429. Periksa blok qos/ratelimit/router pada /api/v1/auth/login"

step "2) Bukti header mentah respons 429 (curl -i)"
curl -s -i -m 10 -X POST "$BASE/auth/login" -H "$JSON" -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" | head -n 14 | sed 's/^/     /'
hdr="$(curl -s -i -m 10 -X POST "$BASE/auth/login" -H "$JSON" -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" | tr -d '\r' | grep -iE '^(retry-after|x-ratelimit[a-z-]*|ratelimit[a-z-]*):')"
if [ -n "$hdr" ]; then ok "header kontrol laju terdeteksi: $(echo "$hdr" | tr '\n' ' ')"
else warn "TIDAK ada header Retry-After / X-RateLimit-* pada respons 429. KrakenD CE tidak selalu menambahkannya; lihat docs/tugas5-api-gateway.md bagian 'Header Retry-After' untuk opsi tambalan."; fi

step "3) Pembatasan hanya berlaku per rute: rute lain tetap melayani (isolasi kebijakan)"
c="$(code_of "$BASE/catalog/products/$PRODUCT_ID")"; [ "$c" = "200" ] && ok "GET /catalog/products/{id} tetap HTTP 200 saat login sedang dibatasi" || bad "rute lain ikut terdampak (HTTP $c)"

step "4) Pemulihan: token terisi ulang 1 tiap 6 detik"
info "menunggu 7 detik ..."; sleep 7
c="$(curl -s -o /dev/null -m 10 -w '%{http_code}' -X POST "$BASE/auth/login" -H "$JSON" -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")"
[ "$c" = "200" ] && ok "setelah menunggu, permintaan diterima lagi (HTTP $c)" || warn "masih HTTP $c (tunggu lebih lama atau reset dengan docker compose restart api-gateway)"
finish
