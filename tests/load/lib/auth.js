// Shared auth helpers for CareerOS k6 load tests.
import http from 'k6/http';
import { check } from 'k6';

export const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
export const LOAD_EMAIL = __ENV.LOAD_EMAIL || 'loadtest@gram.edu';
export const LOAD_PASSWORD = __ENV.LOAD_PASSWORD || 'loadtest-pass-12345';

export function ensureAuthToken() {
  const loginRes = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    JSON.stringify({ email: LOAD_EMAIL, password: LOAD_PASSWORD }),
    { headers: { 'Content-Type': 'application/json' } },
  );

  if (loginRes.status === 200) {
    const body = loginRes.json();
    return body.token;
  }

  const registerRes = http.post(
    `${BASE_URL}/api/v1/auth/register`,
    JSON.stringify({ email: LOAD_EMAIL, password: LOAD_PASSWORD }),
    { headers: { 'Content-Type': 'application/json' } },
  );

  check(registerRes, {
    'register status 201': (r) => r.status === 201,
  });

  const body = registerRes.json();
  return body.token;
}

export function authHeaders(token) {
  return {
    Authorization: `Bearer ${token}`,
    'Content-Type': 'application/json',
  };
}
