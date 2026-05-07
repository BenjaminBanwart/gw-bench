import http from 'k6/http';
import { check, sleep } from 'k6';

const TARGET_URL = __ENV.TARGET_URL || 'http://localhost:8080/sse?count=10&interval_ms=100';
const SUMMARY_PATH = __ENV.K6_SUMMARY_PATH || '/tmp/k6-summary.json';

export const options = {
  vus: parseInt(__ENV.VUS || '50'),
  duration: __ENV.DURATION || '60s',
  thresholds: {
    http_req_failed: ['rate<0.05'],
  },
};

export default function () {
  const res = http.get(TARGET_URL, {
    timeout: '30s',
  });

  check(res, {
    'status is 200': (r) => r.status === 200,
    'content-type is event-stream': (r) =>
      r.headers['Content-Type'] &&
      r.headers['Content-Type'].includes('text/event-stream'),
    'body contains events': (r) => r.body && r.body.includes('event: message'),
  });

  // Brief pause between stream connections
  sleep(Math.random() * 0.2);
}

export function handleSummary(data) {
  return {
    [SUMMARY_PATH]: JSON.stringify(data),
  };
}
