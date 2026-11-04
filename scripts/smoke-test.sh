#!/usr/bin/env bash
# Uji alur lintas service: checkout -> faktur (Payment) -> pengiriman (Logistics) -> bayar -> status order.
# Jalankan setelah `docker compose up --build -d` dan tunggu sekitar 30 detik.
set -uo pipefail

BASE="${BASE:-http://localhost:3000/api/v1}"
EMAIL="${EMAIL:-wayan.putra@gmail.com}"
PASSWORD="${PASSWORD:-password123}"
PRODUCT_ID="${PRODUCT_ID:-p0000001-0000-0000-0000-000000000001}"
ADDRESS_ID="${ADDRESS_ID:-a0000001-0000-0000-0000-000000000001}"
WAREHOUSE_ID="${WAREHOUSE_ID:-w0000001-0000-0000-0000-000000000001}"
PAYMENT_METHOD_ID="${PAYMENT_METHOD_ID:-pm000001-0000-0000-0000-000000000001}" # QRIS
WAIT="${WAIT:-8}"

# jget '<json>' 'a.b.c' -> nilai pada path (kosong bila tidak ada)
jget() {
  python3 -c "
import sys, json
try:
    v = json.loads(sys.argv[1])
    for k in sys.argv[2].split('.'):
        v = v[k]
    print(v)
except Exception:
    print('')
" "$1" "$2"
}
check() { # check <nama> <aktual> <diharapkan>
  if [ "$2" = "$3" ]; then echo "  [PASS] $1: $2"; else echo "  [FAIL] $1: dapat '$2', seharusnya '$3'"; FAILED=1; fi
}
FAILED=0
JSON='Content-Type: application/json'

echo "1) Login sebagai $EMAIL (auth-service)"
RESP=$(curl -s -X POST "$BASE/auth/login" -H "$JSON" -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
TOKEN=$(jget "$RESP" data.token)
[ -n "$TOKEN" ] || { echo "  [FAIL] login gagal: $RESP"; exit 1; }
AUTH="Authorization: Bearer $TOKEN"
echo "  [PASS] token diterima"

echo "2) Tambah ke keranjang (order-service -> catalog-service lewat gRPC) lalu checkout"
curl -s -o /dev/null -X POST "$BASE/orders/cart" -H "$AUTH" -H "$JSON" -d "{\"product_id\":\"$PRODUCT_ID\",\"quantity\":1}"
RESP=$(curl -s -X POST "$BASE/orders/checkout" -H "$AUTH" -H "$JSON" \
  -d "{\"shipping_address_id\":\"$ADDRESS_ID\",\"warehouse_id\":\"$WAREHOUSE_ID\"}")
ORDER_ID=$(jget "$RESP" data.id)
ORDER_NO=$(jget "$RESP" data.order_number)
[ -n "$ORDER_ID" ] || { echo "  [FAIL] checkout gagal: $RESP"; exit 1; }
echo "  [PASS] order $ORDER_NO dibuat ($ORDER_ID)"

echo "   ...menunggu ${WAIT} detik (outbox -> RabbitMQ -> consumer, sinkron replika)"
sleep "$WAIT"

echo "3) Faktur dibuat otomatis oleh payment-service (event OrderPlaced)"
RESP=$(curl -s "$BASE/payments/invoices/order/$ORDER_ID" -H "$AUTH")
check "status faktur" "$(jget "$RESP" data.payment_status)" "UNPAID"

echo "4) Pengiriman dibuat otomatis oleh logistics-service (event OrderPlaced)"
RESP=$(curl -s "$BASE/logistics/order/$ORDER_ID" -H "$AUTH")
check "status pengiriman" "$(jget "$RESP" data.shipping_order.current_status)" "READY_FOR_PICKUP"

echo "5) Bayar (payment-service -> order-service lewat HTTP internal)"
curl -s -o /dev/null -X POST "$BASE/payments/pay" -H "$AUTH" -H "$JSON" \
  -d "{\"order_id\":\"$ORDER_ID\",\"payment_method_id\":\"$PAYMENT_METHOD_ID\"}"
sleep 1
RESP=$(curl -s "$BASE/orders/$ORDER_ID" -H "$AUTH")
check "status order" "$(jget "$RESP" data.order.status_code)" "PROCESSING"

echo
if [ "$FAILED" = "0" ]; then echo "SEMUA PENGUJIAN LULUS"; else echo "ADA PENGUJIAN GAGAL (lihat log: docker compose logs --tail=50 <service>)"; exit 1; fi