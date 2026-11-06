// k6 load test:
//   k6 run -e BASE_URL=http://localhost:8080 -e TOKEN=local-dev-token-0123456789 loadtest/api.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 20 },
    { duration: '2m', target: 50 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<500'],
  },
};

const BASE = __ENV.BASE_URL || 'http://localhost:8080';
const TOKEN = __ENV.TOKEN || '';
const HEADERS = { headers: { Authorization: `Bearer ${TOKEN}` } };

export default function () {
  const list = http.get(`${BASE}/api/v1/targets`, HEADERS);
  check(list, { 'list 200': (r) => r.status === 200 });
  const health = http.get(`${BASE}/healthz`);
  check(health, { 'health 200': (r) => r.status === 200 });
  sleep(0.2);
}
