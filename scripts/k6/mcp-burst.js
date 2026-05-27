import http from 'k6/http';
import { check, sleep } from 'k6';

const TARGET_URL = __ENV.TARGET_URL || 'http://localhost:8080/mcp';
const SUMMARY_PATH = __ENV.K6_SUMMARY_PATH || '/tmp/k6-summary.json';

export const options = {
  vus: parseInt(__ENV.VUS || '50'),
  duration: __ENV.DURATION || '60s',
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(50)', 'p(95)', 'p(99)', 'p(99.9)'],
  thresholds: {
    http_req_failed: ['rate<0.05'],
  },
};

// MCP JSON-RPC request helpers
function initializeRequest(id) {
  return JSON.stringify({
    jsonrpc: '2.0',
    method: 'initialize',
    id: id,
    params: {
      protocolVersion: '2025-03-26',
      capabilities: {},
      clientInfo: { name: 'gw-bench-k6', version: '0.1.0' },
    },
  });
}

function toolsListRequest(id) {
  return JSON.stringify({
    jsonrpc: '2.0',
    method: 'tools/list',
    id: id,
    params: {},
  });
}

function toolsCallRequest(id, toolName, args) {
  return JSON.stringify({
    jsonrpc: '2.0',
    method: 'tools/call',
    id: id,
    params: {
      name: toolName,
      arguments: args || {},
    },
  });
}

const headers = { 'Content-Type': 'application/json' };

export default function () {
  // Initialize session
  let res = http.post(TARGET_URL, initializeRequest(1), { headers });
  check(res, { 'initialize 200': (r) => r.status === 200 });

  // Extract session ID from Mcp-Session header if present
  const sessionHeaders = Object.assign({}, headers);
  const sessionId = res.headers['Mcp-Session'];
  if (sessionId) {
    sessionHeaders['Mcp-Session'] = sessionId;
  }

  // List tools
  res = http.post(TARGET_URL, toolsListRequest(2), { headers: sessionHeaders });
  check(res, { 'tools/list 200': (r) => r.status === 200 });

  // Call echo tool
  res = http.post(TARGET_URL, toolsCallRequest(3, 'echo', { message: 'hello from gw-bench' }), { headers: sessionHeaders });
  check(res, { 'tools/call echo 200': (r) => r.status === 200 });

  // Call delay tool (short delay)
  res = http.post(TARGET_URL, toolsCallRequest(4, 'delay', { ms: 10 }), { headers: sessionHeaders });
  check(res, { 'tools/call delay 200': (r) => r.status === 200 });

  // Simulate realistic idle time between bursts
  sleep(Math.random() * 0.5);
}

export function handleSummary(data) {
  return {
    [SUMMARY_PATH]: JSON.stringify(data),
  };
}
