#!/usr/bin/env bash
# Menerbitkan N transaksi checkout lewat API (setiap checkout = 1 event OrderPlaced).
# Jeda SLEEP detik diperlukan karena nomor order/invoice di kode dibuat dari time.Now().Unix()%10000 (UNIQUE).
set -euo pipefail

BASE="${BASE:-http://localhost:3000/api/v1}"
COUNT="${COUNT:-50}"
SLEEP="${SLEEP:-1.1}"
EMAIL="${EMAIL:-wayan.putra@gmail.com}"
PASSWORD="${PASSWORD:-password123}"
PRODUCT_ID="${PRODUCT_ID:-p0000001-0000-0000-0000-000000000001}"
ADDRESS_ID="${ADDRESS_ID:-a0000001-0000-0000-0000-000000000001}"
WAREHOUSE_ID="${WAREHOUSE_ID:-w0000001-0000-0000-0000-000000000001}"

TOKEN=$(curl -s -X POST "$BASE/auth/login" -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['token'])")
echo "Login OK. Menjalankan $COUNT checkout (produk $PRODUCT_ID, gudang $WAREHOUSE_ID)..."

ok=0; fail=0
for i in $(seq 1 "$COUNT"); do
  curl -s -o /dev/null -X POST "$BASE/orders/cart" \
    -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
    -d "{\"product_id\":\"$PRODUCT_ID\",\"quantity\":1}"

  code=$(curl -s -o /tmp/checkout_resp.json -w '%{http_code}' -X POST "$BASE/orders/checkout" \
    -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
    -d "{\"shipping_address_id\":\"$ADDRESS_ID\",\"warehouse_id\":\"$WAREHOUSE_ID\"}")

  if [ "$code" = "201" ]; then ok=$((ok+1)); else fail=$((fail+1)); cat /tmp/checkout_resp.json; echo; fi
  echo "[$i/$COUNT] checkout HTTP $code"
  sleep "$SLEEP"
done
echo "Selesai. Sukses: $ok, Gagal: $fail"