#!/usr/bin/env python3
"""Seed the database with test data.

Zero dependencies — Python stdlib only.

Usage:
    python3 scripts/seed.py                  # default test data
    API_URL=http://localhost:9090 python3 scripts/seed.py
    python3 scripts/seed.py --clean          # drop test data first

Requires a running API (make dev).
"""

import json
import os
import sys
import urllib.request
import urllib.error
import subprocess

API_URL = os.environ.get("API_URL", "http://localhost:8080")

# ── Test Data ────────────────────────────────────────────────────────────────

ADMIN = {
    "username": "admin",
    "email": "admin@dotslashstream.local",
    "password": "admin123",
}

USERS = [
    {
        "username": "alice",
        "email": "alice@dotslashstream.local",
        "password": "alice123",
    },
    {
        "username": "bob",
        "email": "bob@dotslashstream.local",
        "password": "bob123",
    },
    {
        "username": "charlie",
        "email": "charlie@dotslashstream.local",
        "password": "charlie123",
    },
]


# ── HTTP Helpers ─────────────────────────────────────────────────────────────

def api(method, path, body=None, headers=None):
    """Make API request. Returns (status, parsed_body)."""
    url = f"{API_URL}{path}"
    data = json.dumps(body).encode() if body else None
    hdrs = {"Content-Type": "application/json"}
    if headers:
        hdrs.update(headers)
    req = urllib.request.Request(url, data=data, headers=hdrs, method=method)
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            return resp.status, json.loads(resp.read())
    except urllib.error.HTTPError as e:
        try:
            return e.code, json.loads(e.read())
        except Exception:
            return e.code, {}


def check_api():
    """Verify API is reachable."""
    try:
        status, _ = api("GET", "/")
        return status < 500
    except Exception:
        return False


# ── Database Helpers ─────────────────────────────────────────────────────────

def psql(query):
    """Run a SQL query against the database via docker compose."""
    result = subprocess.run(
        ["docker", "compose", "exec", "-T", "database", "psql",
         "-U", "postgres", "-d", "dotslashstream", "-c", query],
        capture_output=True, text=True, timeout=10,
    )
    if result.returncode != 0:
        print(f"  ✗ SQL error: {result.stderr.strip()}")
        return False
    return True


def clean():
    """Remove all test data."""
    print("\nCleaning test data...")
    psql("DELETE FROM invites;")
    psql("DELETE FROM users;")
    print("  ✓ Database cleared")


# ── Seed Logic ───────────────────────────────────────────────────────────────

def login_user(user):
    """Return token response for user credentials, or None when unavailable."""
    status, body = api("POST", "/auth/login", {
        "username": user["username"],
        "password": user["password"],
    })
    return body if status == 200 else None


def register_user(user):
    """Register a user. Returns True only when account creation succeeds."""
    status, body = api("POST", "/auth/register", user)
    if status == 201:
        return True
    print(f"  ✗ Register {user['username']}: {status} {body}")
    return False


def create_invite(headers, max_uses):
    """Create a 30-day invite. Returns its token, or None on failure."""
    status, body = api("POST", "/auth/invite/generate", {
        "ttl": "720h",
        "max_uses": max_uses,
    }, headers=headers)
    if status != 201:
        print(f"  ✗ Failed to generate invite: {status} {body}")
        return None
    return body["token"]


def seed():
    """Insert reusable test data, including an admin account."""
    if not check_api():
        print(f"✗ Cannot reach API at {API_URL}")
        print("  Start the API first: make dev")
        sys.exit(1)

    print(f"Seeding test data → {API_URL}\n")

    # 1. Try login first; if no admin exists, create via init route.
    print("Ensuring admin...")
    tokens = login_user(ADMIN)
    if tokens is None:
        # System not initialized — create first admin via /auth/register/admin
        status, body = api("POST", "/auth/register/admin", ADMIN)
        if status == 201:
            print(f"  ✓ {ADMIN['username']} ({ADMIN['email']}) created via init route")
            tokens = body
        elif status == 404:
            # Already initialized but credentials wrong
            print(f"  ✗ Init route unavailable; verify admin credentials")
            sys.exit(1)
        else:
            print(f"  ✗ Failed to create admin: {status} {body}")
            sys.exit(1)
    else:
        print(f"  ✓ {ADMIN['username']} already exists")

    admin_headers = {"Authorization": f"Bearer {tokens['access_token']}"}
    print("  ✓ Admin ready")

    # 2. Determine whether ordinary accounts need an invite.
    status, current = api("GET", "/settings", headers=admin_headers)
    if status != 200:
        print(f"  ✗ Failed to read settings: {status} {current}")
        sys.exit(1)
    signup_open = current["allow_signup_without_invite"]

    print("Registering test users...")
    missing_users = [user for user in USERS if login_user(user) is None]
    invite_token = None
    if missing_users and not signup_open:
        invite_token = create_invite(admin_headers, len(missing_users))
        if invite_token is None:
            sys.exit(1)

    for user in USERS:
        if login_user(user) is not None:
            print(f"  ✓ {user['username']} already exists")
            continue
        payload = {**user}
        if invite_token is not None:
            payload["invite"] = invite_token
        if register_user(payload):
            print(f"  ✓ {user['username']} ({user['email']})")
        else:
            print(f"  ✗ Failed to register {user['username']}")

    # 3. Generate a shareable invite for manual testing.
    print("Generating invite...")
    invite_token = create_invite(admin_headers, 10)
    if invite_token is not None:
        print("  ✓ Invite token (30d, 10 uses)")
        print(f"    {invite_token[:50]}...")

    # Summary
    print("\n" + "─" * 50)
    print(" Test Data Summary")
    print("─" * 50)
    print(f" Admin:     {ADMIN['username']} / {ADMIN['password']}")
    for user in USERS:
        print(f" User:      {user['username']} / {user['password']}")
    print("─" * 50)
    print("\nAll test users share a common password pattern: <name>123")


if __name__ == "__main__":
    if "--clean" in sys.argv:
        clean()
    seed()
