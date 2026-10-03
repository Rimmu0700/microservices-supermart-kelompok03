import http from 'k6/http';
import { check } from 'k6';

const VUS = parseInt(__ENV.VUS || '50');
const DURATION = __ENV.DURATION || '30s';
const BASE = __ENV.BASE_URL || 'http://localhost:3000';
const PRODUCT_ID = __ENV.PRODUCT_ID || 'p0000001-0000-0000-0000-000000000001';
// default: endpoint ringkas (padanan gRPC). Untuk baseline bawaan:
//   -e REST_PATH=/api/v1/catalog/products/<ID>
const PATH = __ENV.REST_PATH || `/api/v1/catalog/products/${PRODUCT_ID}/summary`;

export const options = {
    vus: VUS,
    duration: DURATION,
    summaryTrendStats: ['avg', 'min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
};

export default function () {
    const res = http.get(BASE + PATH);
    check(res, { 'status 200': (r) => r.status === 200 });
}