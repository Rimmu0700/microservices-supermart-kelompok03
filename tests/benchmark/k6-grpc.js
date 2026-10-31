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

let connected = false;

export default function () {
    // satu koneksi per VU, dipakai ulang (HTTP/2 persisten); dicoba lagi bila gagal
    if (!connected) {
        client.connect(ADDR, { plaintext: true, timeout: '10s' });
        connected = true;
    }
    const res = client.invoke('supermart.catalog.v1.CatalogService/GetProduct', { product_id: PRODUCT_ID });
    check(res, { 'status OK': (r) => r && r.status === grpc.StatusOK });
}