import http from 'k6/http';
import { check, sleep } from 'k6';

const TARGET_URL = __ENV.TARGET_URL || 'http://localhost:8080/route';
const ROUTE_COUNT = parseInt(__ENV.ROUTE_COUNT || '500');
const SUMMARY_PATH = __ENV.K6_SUMMARY_PATH || '/tmp/k6-summary.json';

export const options = {
  vus: parseInt(__ENV.VUS || '100'),
  duration: __ENV.DURATION || '60s',
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(50)', 'p(95)', 'p(99)', 'p(99.9)'],
  thresholds: {
    http_req_failed: ['rate<0.05'],
  },
};

// Pre-build route URLs to avoid string allocation in the hot loop.
// Expects the gateway to have /route/001 ... /route/{ROUTE_COUNT} configured.
const routeURLs = [];
for (let i = 1; i <= ROUTE_COUNT; i++) {
  routeURLs.push(`${TARGET_URL}/${String(i).padStart(3, '0')}`);
}

export default function () {
  const idx = Math.floor(Math.random() * routeURLs.length);
  const res = http.get(routeURLs[idx]);

  check(res, {
    'status is 200': (r) => r.status === 200,
  });

  sleep(Math.random() * 0.01);
}

export function handleSummary(data) {
  return {
    [SUMMARY_PATH]: JSON.stringify(data),
  };
}
