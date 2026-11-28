#!/usr/bin/env bash
# =====================================================================================
# NAMA       : 02-uji-agregasi-dan-degradasi-parsial.sh
# KATEGORI   : PEMBUKTIAN + SIMULASI GANGGUAN (Tugas 5 - Bagian II.2 dan II.3)
# FUNGSI     : Menguji endpoint agregasi paralel (storefront dan order-summary), optimasi payload, serta graceful degradation saat service mati dan saat service macet (timeout).
# DIJALANKAN : bash scripts/gateway/02-uji-agregasi-dan-degradasi-parsial.sh
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
TH="${TMPDIR:-/tmp}/_t5_h.txt"; TB="${TMPDIR:-/tmp}/_t5_b.txt"
trap 'docker unpause logistics-service >/dev/null 2>&1; docker start logistics-service >/dev/null 2>&1' EXIT
title "TUGAS 5 - UJI 2: AGREGASI PARALEL (BFF) DAN GRACEFUL DEGRADATION"
require_gateway
need_order && ok "id order untuk pengujian: $ORDER_ID" || { bad "tidak ada order. Jalankan bash scripts/smoke-test.sh lalu ulangi"; exit 1; }
AUTHH=(-H "Authorization: Bearer $TOKEN")

step "1) Endpoint agregasi publik /api/v1/storefront (produk + kategori dari catalog, flash sale dari promotion)"
curl -s -m 10 -D "$TH" -o "$TB" "$BASE/storefront"
k="$(keys_of "$(cat "$TB")")"; info "kunci pada JSON gabungan: $k   | ukuran: $(wc -c < "$TB" | tr -d ' ') bytes"
n=0; for g in products categories flash_sales; do case " $k " in *" $g "*) n=$((n+1));; esac; done
[ "$n" -ge 2 ] && ok "satu respons memuat gabungan $n sumber dari 2 service berbeda (catalog + promotion)" || bad "agregasi storefront tidak lengkap ($k)"

step "2) Endpoint agregasi /api/v1/order-summary/{order_id} (order + faktur + pengiriman, 3 service paralel)"
curl -s -m 10 -D "$TH" -o "$TB" "$BASE/order-summary/$ORDER_ID" "${AUTHH[@]}"
head -n 1 "$TH" | tr -d '\r' | sed 's/^/     /'; grep -i '^x-krakend' "$TH" | tr -d '\r' | sed 's/^/     /'
"$PY" -c "
import json,sys
d=json.load(open(sys.argv[1]))
print(json.dumps(d,indent=2,ensure_ascii=False)[:900])
" "$TB" | sed 's/^/     /'
k="$(keys_of "$(cat "$TB")")"; info "kunci tingkat atas: $k"
n=0; for g in order_info invoice_info delivery_info; do case " $k " in *" $g "*) n=$((n+1));; esac; done
[ "$n" -ge 2 ] && ok "JSON gabungan memuat $n kelompok data dari service berbeda" || bad "agregasi order-summary hanya $n kelompok"

step "3) Optimasi payload: ukuran order mentah vs bagian order_info pada agregasi"
raw="$(curl -s -m 10 "$BASE/orders/$ORDER_ID" "${AUTHH[@]}" | wc -c | tr -d ' ')"
opt="$("$PY" -c "
import json,sys
d=json.load(open(sys.argv[1])); print(len(json.dumps(d.get('order_info',{}),separators=(',',':'))))
" "$TB")"
info "GET /orders/{id} mentah: $raw bytes | order_info setelah deny[histories,notes,shipping]: $opt bytes"
[ "${opt:-0}" -lt "${raw:-0}" ] && ok "payload order_info lebih ringkas ($((100-opt*100/raw))% lebih kecil)" || warn "tidak ada penghematan (data order mungkin tidak memiliki atribut yang disaring)"

step "4) GRACEFUL DEGRADATION (a): logistics-service MATI -> gateway tetap menjawab data parsial"
docker stop logistics-service >/dev/null 2>&1 && ok "logistics-service dimatikan" || bad "gagal mematikan logistics-service"
sleep 2
T0=$(date +%s%N 2>/dev/null || echo 0)
curl -s -m 10 -D "$TH" -o "$TB" "$BASE/order-summary/$ORDER_ID" "${AUTHH[@]}"
T1=$(date +%s%N 2>/dev/null || echo 0)
st="$(head -1 "$TH" | awk '{print $2}')"; comp="$(grep -i '^x-krakend-completed' "$TH" | tr -d '\r')"
k="$(keys_of "$(cat "$TB")")"
head -n 8 "$TH" | tr -d '\r' | sed 's/^/     /'; info "kunci: $k"; [ -n "$comp" ] && info "penanda parsial: $comp"
[ "$T0" != "0" ] && info "waktu respons: $(( (T1-T0)/1000000 )) ms (service mati = koneksi langsung ditolak, jawaban cepat)"
if [ "$st" = "200" ]; then ok "gateway tetap HTTP 200"; else bad "status $st (harapan 200)"; fi
case " $k " in *" order_info "*) ok "data order_info (service hidup) tetap tersaji";; *) bad "order_info hilang";; esac
case " $k " in *" delivery_info "*) warn "delivery_info masih ada (container belum benar-benar mati?)";; *) ok "delivery_info hilang (sesuai: logistics mati)";; esac
docker start logistics-service >/dev/null 2>&1 && ok "logistics-service dinyalakan kembali"

step "5) GRACEFUL DEGRADATION (b): logistics-service MACET (docker pause) -> gateway menunggu sampai TIMEOUT lalu data parsial"
docker pause logistics-service >/dev/null 2>&1 && ok "logistics-service dibekukan (menerima koneksi tetapi tidak menjawab)" || warn "gagal pause (lewati)"
sleep 1
T0=$(date +%s%N 2>/dev/null || echo 0)
curl -s -m 15 -D "$TH" -o "$TB" "$BASE/order-summary/$ORDER_ID" "${AUTHH[@]}"
T1=$(date +%s%N 2>/dev/null || echo 0)
st="$(head -1 "$TH" | awk '{print $2}')"; k="$(keys_of "$(cat "$TB")")"; comp="$(grep -i '^x-krakend-completed' "$TH" | tr -d '\r')"
ms=$(( (T1-T0)/1000000 )); info "status: $st | kunci: $k | ${comp:-tanpa header completed} | waktu respons: ${ms} ms"
if [ "$st" = "200" ] && [ "$ms" -ge 2000 ] && [ "$ms" -le 4000 ]; then ok "gateway menunggu batas waktu endpoint (2500 ms) lalu mengembalikan data parsial ($ms ms)"; else warn "perilaku timeout berbeda dari harapan (status $st, $ms ms); lihat nilai 'timeout' pada endpoint di gateway/routes/order.json"; fi
docker unpause logistics-service >/dev/null 2>&1; docker start logistics-service >/dev/null 2>&1; ok "logistics-service dipulihkan"
finish
