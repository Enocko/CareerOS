import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Rate, Counter } from 'k6/metrics';
import { BASE_URL, ensureAuthToken, authHeaders } from './lib/auth.js';

const browseLatency = new Trend('careeros_browse_latency', true);
const searchLatency = new Trend('careeros_search_latency', true);
const recommendLatency = new Trend('careeros_recommend_latency', true);
const applicationsLatency = new Trend('careeros_applications_latency', true);
const notificationsLatency = new Trend('careeros_notifications_latency', true);
const errorRate = new Rate('careeros_errors');
const requests = new Counter('careeros_requests');

export const options = {
  scenarios: {
    baseline: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 1 },
        { duration: '30s', target: 10 },
        { duration: '30s', target: 25 },
        { duration: '30s', target: 50 },
        { duration: '30s', target: 100 },
        { duration: '30s', target: 0 },
      ],
      gracefulRampDown: '10s',
    },
  },
  thresholds: {
    careeros_errors: ['rate<0.05'],
  },
};

let token;

export function setup() {
  token = ensureAuthToken();
  return { token };
}

export default function (data) {
  const headers = authHeaders(data.token);

  const browseRes = http.get(`${BASE_URL}/api/v1/opportunities?per_page=20&page=1`, { headers });
  requests.add(1);
  browseLatency.add(browseRes.timings.duration);
  errorRate.add(browseRes.status >= 500);
  check(browseRes, { 'browse 200': (r) => r.status === 200 });

  const searchRes = http.get(
    `${BASE_URL}/api/v1/opportunities?query=engineer&category=internship&per_page=20`,
    { headers },
  );
  requests.add(1);
  searchLatency.add(searchRes.timings.duration);
  errorRate.add(searchRes.status >= 500);
  check(searchRes, { 'search 200': (r) => r.status === 200 });

  const recRes = http.get(`${BASE_URL}/api/v1/opportunities/recommended?per_page=20`, { headers });
  requests.add(1);
  recommendLatency.add(recRes.timings.duration);
  errorRate.add(recRes.status >= 500);
  check(recRes, { 'recommended 200': (r) => r.status === 200 });

  const appsRes = http.get(`${BASE_URL}/api/v1/applications`, { headers });
  requests.add(1);
  applicationsLatency.add(appsRes.timings.duration);
  errorRate.add(appsRes.status >= 500);
  check(appsRes, { 'applications 200': (r) => r.status === 200 });

  const notifRes = http.get(`${BASE_URL}/api/v1/notifications/unread-count`, { headers });
  requests.add(1);
  notificationsLatency.add(notifRes.timings.duration);
  errorRate.add(notifRes.status >= 500);
  check(notifRes, { 'notifications 200': (r) => r.status === 200 });

  sleep(0.2);
}

export function handleSummary(data) {
  return {
    stdout: textSummary(data),
    '/scripts/results/baseline-summary.json': JSON.stringify(data, null, 2),
  };
}

function textSummary(data) {
  const lines = ['CareerOS baseline load summary'];
  for (const name of [
    'http_req_duration',
    'careeros_browse_latency',
    'careeros_search_latency',
    'careeros_recommend_latency',
    'careeros_applications_latency',
    'careeros_notifications_latency',
  ]) {
    const metric = data.metrics[name];
    if (!metric || !metric.values) continue;
    const p50 = metric.values.med ?? metric.values['p(50)'];
    const p95 = metric.values['p(95)'];
    const p99 = metric.values['p(99)'];
    lines.push(`${name}: p50=${p50}ms p95=${p95}ms p99=${p99}ms`);
  }
  if (data.metrics.careeros_errors) {
    lines.push(`error_rate=${data.metrics.careeros_errors.values.rate}`);
  }
  if (data.metrics.careeros_requests) {
    lines.push(`requests=${data.metrics.careeros_requests.values.count}`);
  }
  return lines.join('\n') + '\n';
}
