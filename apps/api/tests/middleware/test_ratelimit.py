"""E2E tests for the global rate-limit middleware.

The rate limiter enforces a sliding-window limit per RemoteAddr using
Redis sorted sets.  All requests from this test runner share one IP,
so these tests drain the budget once and verify the 429 responses.

Run in isolation — these tests consume the shared rate-limit bucket:
    python3 -m tests middleware/test_ratelimit.py
"""

import json
import time
import urllib.request
import urllib.error

from tests.runner import test, section, assert_eq, assert_in, assert_true
from tests.client import API_URL

section("Rate Limiting")

LIMIT = 100  # must match internal/middleware/chain.go


def _raw_request(method, path, body=None, headers=None):
    """Make a raw HTTP request. Returns (status_code, parsed_json)."""
    url = f"{API_URL}{path}"
    data = json.dumps(body).encode() if body is not None else None
    hdrs = {"Content-Type": "application/json"}
    if headers:
        hdrs.update(headers)
    req = urllib.request.Request(url, data=data, headers=hdrs, method=method)
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            return resp.status, json.loads(resp.read())
    except urllib.error.HTTPError as e:
        body_out = {}
        try:
            body_out = json.loads(e.read())
        except Exception:
            pass
        return e.code, body_out


def _drain_budget(requests=LIMIT + 10):
    """Send requests until rate-limited or budget exhausted.

    Sleeps briefly between requests so each lands in a unique millisecond,
    which prevents sorted-set member collisions in the Redis driver.
    """
    for i in range(requests):
        status, body = _raw_request("GET", "/")
        if status == 429:
            return status, body
        time.sleep(0.002)  # 2ms gap → unique sorted-set member per request
    return None, None


# ── Drain first so all subsequent tests see a 429 ────────────────────────────


@test("Exhausting the limit returns 429")
def test_01_drain_and_verify_429():
    """Send enough requests to exhaust the window, then confirm 429."""
    status, body = _drain_budget()
    assert_eq(status, 429, "rate limit was not triggered after draining")
    assert_in("error", body)
    assert_eq(body["error"], "rate limit exceeded")  # pyright: ignore[reportOptionalSubscript]


# ── These run after the drain and should all get 429 ──────────────────────────


@test("429 body contains only the error field")
def test_02_429_body_shape():
    status, body = _raw_request("GET", "/")
    assert_eq(status, 429)
    assert_eq(set(body.keys()), {"error"})


@test("429 error message is 'rate limit exceeded'")
def test_03_429_error_message():
    status, body = _raw_request("GET", "/")
    assert_eq(status, 429)
    assert_eq(body["error"], "rate limit exceeded")


@test("429 response has JSON content-type")
def test_04_429_content_type():
    url = f"{API_URL}/"
    req = urllib.request.Request(url, method="GET")
    try:
        urllib.request.urlopen(req, timeout=10)
    except urllib.error.HTTPError as e:
        assert_eq(e.code, 429)
        ct = e.headers.get("Content-Type", "")
        assert_true(
            "application/json" in ct,
            f"expected application/json content-type, got {ct!r}",
        )
        return
    assert_true(False, "expected 429 but got 200")


@test("POST requests are also rate-limited")
def test_05_post_rate_limited():
    status, body = _raw_request(
        "POST",
        "/auth/login",
        {
            "username": "doesntmatter",
            "password": "doesntmatter",
        },
    )
    assert_eq(status, 429)
    assert_eq(body["error"], "rate limit exceeded")


@test("Non-existent routes are still rate-limited")
def test_06_nonexistent_route_rate_limited():
    status, body = _raw_request("GET", "/this-does-not-exist-abc123")
    assert_eq(status, 429)
    assert_in("error", body)
