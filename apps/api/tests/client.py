"""Shared HTTP client and helpers for e2e tests."""

import json
import os
import uuid
import urllib.request
import urllib.error

API_URL = os.environ.get("API_URL", "http://localhost:8080")


class APIError(Exception):
    def __init__(self, status, body):
        self.status = status
        self.body = body
        super().__init__(f"HTTP {status}: {body}")


def request(method, path, body=None, headers=None):
    """Make an HTTP request. Returns (status_code, parsed_json)."""
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
        resp_body = {}
        try:
            resp_body = json.loads(e.read())
        except Exception:
            pass
        return e.code, resp_body


def raw_request(method, path, data=None, headers=None):
    """Make a raw HTTP request (no JSON encoding). Returns (status_code, body_bytes)."""
    url = f"{API_URL}{path}"
    hdrs = headers or {}
    req = urllib.request.Request(url, data=data, headers=hdrs, method=method)
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            return resp.status, resp.read()
    except urllib.error.HTTPError as e:
        return e.code, e.read()


def auth_headers(token):
    return {"Authorization": f"Bearer {token}"}


def unique_user():
    """Generate unique user credentials."""
    uid = uuid.uuid4().hex
    return {
        "username": f"test{uid}",
        "email": f"test_{uid}@e2e.test",
        "password": f"pass_{uid}",
    }


def register_and_login(user=None):
    """Register a user and return (user_dict, access_token, refresh_token)."""
    if user is None:
        user = unique_user()
    status, body = request("POST", "/auth/register", user)
    assert status == 201, f"register failed: {status} {body}"
    return user, body["access_token"], body["refresh_token"]
