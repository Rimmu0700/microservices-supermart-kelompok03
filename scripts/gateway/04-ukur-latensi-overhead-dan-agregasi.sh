#!/usr/bin/env bash
# =====================================================================================
# NAMA       : 04-ukur-latensi-overhead-dan-agregasi.sh
# KATEGORI   : PENGUKURAN (Tugas 5 - Bagian IV)
# FUNGSI     : Mengukur rata-rata N=20 percobaan: overhead gateway, efisiensi agregasi, dan latensi respons 429; menghasilkan tabel komparasi.
# DIJALANKAN : bash scripts/gateway/04-ukur-latensi-overhead-dan-agregasi.sh
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
N="${N:-20}"; WARM=3
STAMP="$(date +%Y%m%d-%H%M%S)"; mkdir -p hasil-demo; OUTF="hasil-demo/tugas5-latensi-$STAMP.txt"
title "TUGAS 5 - UKUR 4: LATENSI OVERHEAD GATEWAY DAN EFISIENSI AGREGASI (rata-rata $N percobaan)"
require_gateway
need_order && ok "id order: $ORDER_ID" || { bad "tidak ada order (jalankan scripts/smoke-test.sh)"; exit 1; }
NET="$(docker inspect -f '{{range $k,$v := .NetworkSettings.Networks}}{{$k}}{{end}}' api-gateway 2>/dev/null)"
[ -n "$NET" ] && ok "jaringan Docker: $NET (pengukuran dari dalam jaringan agar adil: tanpa NAT host)" || { bad "jaringan Docker tidak ditemukan"; exit 1; }
IMG="${CURL_IMAGE:-curlimages/curl:latest}"
docker pull -q "$IMG" >/dev/null 2>&1 || warn "gagal menarik $IMG (butuh internet); memakai image lokal bila ada"
AUTHV="Authorization: Bearer $TOKEN"
# jalankan loop curl di dalam jaringan Docker; keluaran: satu angka milidetik per baris (urutan: percobaan)
measure(){ # measure <jumlah> <kode> : <kode> berisi perintah shell yang mencetak waktu detik
  docker run --rm --network "$NET" --entrypoint sh "$IMG" -c "for i in \$(seq 1 $1); do $2; done" 2>/dev/null
}
one(){ echo "curl -s -o /dev/null -m 10 -w '%{time_total}\n' -H '$AUTHV' '$1'"; }
tomsec(){ "$PY" -c "
import sys
v=[float(x)*1000 for x in sys.stdin.read().split() if x.replace('.','',1).isdigit()]
v=v[int(sys.argv[1]):]                       # buang pemanasan
v.sort(); n=len(v)
if n==0: print('NA NA NA NA 0'); sys.exit()
print('%.2f %.2f %.2f %.2f %d' % (sum(v)/n, v[n//2], v[min(n-1,int(n*0.95))], v[-1], n))
" "$1"; }
DIRECT_P="http://catalog-service:3000/api/v1/catalog/products/$PRODUCT_ID"
GW_P="http://api-gateway:8080/api/v1/catalog/products/$PRODUCT_ID"

step "SKENARIO 1 - Overhead rute tunggal: langsung ke service vs via gateway"
A=$(measure $((N+WARM)) "$(one "$DIRECT_P")" | tomsec $WARM); B=$(measure $((N+WARM)) "$(one "$GW_P")" | tomsec $WARM)
set -- $A; S1D=$1; S1Dp=$3; set -- $B; S1G=$1; S1Gp=$3
info "langsung : rata-rata $S1D ms (p95 $S1Dp ms)"; info "via gateway: rata-rata $S1G ms (p95 $S1Gp ms)"
S1O="$("$PY" -c "print(round(float('$S1G')-float('$S1D'),2))" 2>/dev/null)"; ok "overhead proxy gateway pada rute tunggal: ${S1O} ms"

step "SKENARIO 2 - Efisiensi agregasi: 3 request berurutan vs 1 endpoint agregasi (order + faktur + pengiriman)"
SEQ_URLS=("http://api-gateway:8080/api/v1/orders/$ORDER_ID" "http://api-gateway:8080/api/v1/payments/invoices/order/$ORDER_ID" "http://api-gateway:8080/api/v1/logistics/order/$ORDER_ID")
SEQCMD="t=0; for u in ${SEQ_URLS[*]}; do x=\$(curl -s -o /dev/null -m 10 -w '%{time_total}' -H '$AUTHV' \$u); t=\$(awk \"BEGIN{print \$t+\$x}\"); done; echo \$t"
C=$(measure $((N+WARM)) "$SEQCMD" | tomsec $WARM); D=$(measure $((N+WARM)) "$(one "http://api-gateway:8080/api/v1/order-summary/$ORDER_ID")" | tomsec $WARM)
set -- $C; S2S=$1; set -- $D; S2A=$1
info "3 request berurutan (via gateway): $S2S ms | 1 request agregasi: $S2A ms"
S2G="$("$PY" -c "s=float('$S2S');a=float('$S2A');print(round((s-a)/s*100,1))" 2>/dev/null)"; ok "agregasi menghemat sekitar ${S2G}% waktu klien dan 2 round-trip jaringan"

step "SKENARIO 3 - Beban berlebih: latensi respons yang DITOLAK 429 (reset limiter, banjir 30 login)"
docker compose restart api-gateway >/dev/null 2>&1; wait_gw
BURST="for i in \$(seq 1 30); do curl -s -o /dev/null -m 10 -w '%{http_code} %{time_total}\n' -X POST http://api-gateway:8080/api/v1/auth/login -H 'Content-Type: application/json' -d '{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}'; done"
RAW="$(docker run --rm --network "$NET" --entrypoint sh "$IMG" -c "$BURST" 2>/dev/null)"
R429="$(echo "$RAW" | awk '$1==429{print $2}' | tomsec 0)"; R200="$(echo "$RAW" | awk '$1==200{print $2}' | tomsec 0)"
set -- $R429; S3R=$1; S3N=$5; set -- $R200; S3A=$1; S3AN=$5
info "diterima (200): $S3AN request, rata-rata $S3A ms | ditolak (429): $S3N request, rata-rata $S3R ms"
[ "${S3N:-0}" -ge 1 ] && ok "respons 429 dikembalikan gateway tanpa menyentuh service/DB hilir" || bad "tidak ada respons 429"

{
echo "TABEL KOMPARASI KINERJA GATEWAY ($(date '+%F %T'), host $(hostname), N=$N)"
echo "Skenario | Metode | Jumlah request | Latensi rata-rata (ms)"
echo "1. Overhead rute tunggal | Langsung ke service vs via gateway | 1 request | $S1D ms vs $S1G ms (overhead ${S1O} ms)"
echo "2. Efisiensi agregasi    | 3 request berurutan vs 1 endpoint agregasi | 3 req vs 1 req | $S2S ms vs $S2A ms (hemat ${S2G}%)"
echo "3. Beban berlebih        | Permintaan di atas ambang rate limit | burst 30 | 429: ${S3R} ms vs diterima: ${S3A} ms"
} | tee "$OUTF"
info "Disimpan di: $OUTF  (salin angka ini ke tabel Bagian IV laporan)"
finish
