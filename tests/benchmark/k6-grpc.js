import grpc from 'k6/net/grpc';
import { check } from 'k6';

const VUS = parseInt(__ENV.VUS || '50');
const DURATION = __ENV.DURATION || '30s';
const ADDR = __ENV.GRPC_ADDR || 'localhost:50051';
const PRODUCT_ID = __ENV.PRODUCT_ID || 'p0000001-0000-0000-0000-000000000001';

const client = new grpc.Client();
client.load(['../../proto'], 'catalog/v1/catalog.proto');

export const options = {
    vus: VUS,
    duration: DURATION,
    summaryTrendStats: ['avg', 'min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
};

export default function () {
    // satu koneksi per VU, dipakai ulang (HTTP/2 persisten)
    if (__ITER === 0) {
        client.connect(ADDR, { plaintext: true });
    }
    const res = client.invoke('supermart.catalog.v1.CatalogService/GetProduct', { product_id: PRODUCT_ID });
    check(res, { 'status OK': (r) => r && r.status === grpc.StatusOK });
}