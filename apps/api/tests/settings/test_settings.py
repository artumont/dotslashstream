import os

from tests.runner import assert_eq, assert_in, assert_true, section, skip, test
from tests.client import auth_headers, raw_request, request, unique_user

section("Settings")

ADMIN_USERNAME = os.environ.get("E2E_ADMIN_USERNAME", "admin")
ADMIN_PASSWORD = os.environ.get("E2E_ADMIN_PASSWORD", "admin123")


def admin_headers():
    status, body = request("POST", "/auth/login", {
        "username": ADMIN_USERNAME,
        "password": ADMIN_PASSWORD,
    })
    if status != 200:
        skip("requires seeded admin; run: make seed")
    return auth_headers(body["access_token"])


def read_settings(headers):
    status, body = request("GET", "/settings", headers=headers)
    assert_eq(status, 200)
    assert_in("allow_signup_without_invite", body)
    assert_true(isinstance(body["allow_signup_without_invite"], bool))
    return body


def update_settings(headers, value):
    status, body = request("PATCH", "/settings", {
        "allow_signup_without_invite": value,
    }, headers=headers)
    assert_eq(status, 200)
    assert_eq(body["allow_signup_without_invite"], value)


@test("Settings endpoints require authentication")
def test_settings_requires_auth():
    status, _ = request("GET", "/settings")
    assert_eq(status, 401)

    status, _ = request("PATCH", "/settings", {
        "allow_signup_without_invite": False,
    })
    assert_eq(status, 401)


@test("Settings endpoints reject non-admin users")
def test_settings_rejects_non_admin():
    user = unique_user()
    status, body = request("POST", "/auth/register", user)
    assert_eq(status, 201)
    headers = auth_headers(body["access_token"])

    status, _ = request("GET", "/settings", headers=headers)
    assert_eq(status, 403)

    status, _ = request("PATCH", "/settings", {
        "allow_signup_without_invite": False,
    }, headers=headers)
    assert_eq(status, 403)


@test("Admin reads initialized settings singleton")
def test_admin_reads_initialized_settings():
    read_settings(admin_headers())


@test("Admin settings rejects malformed and unsupported updates")
def test_admin_rejects_invalid_updates():
    headers = admin_headers()

    status, _ = raw_request("PATCH", "/settings", data=b"not json", headers={
        **headers,
        "Content-Type": "application/json",
    })
    assert_eq(status, 400)

    status, _ = request("PATCH", "/settings", {}, headers=headers)
    assert_eq(status, 400)

    status, _ = request("PATCH", "/settings", {
        "unknown_setting": True,
    }, headers=headers)
    assert_eq(status, 400)

    status, _ = request("PATCH", "/settings", {
        "allow_signup_without_invite": "true",
    }, headers=headers)
    assert_eq(status, 400)

    status, _ = request("PUT", "/settings", {
        "allow_signup_without_invite": True,
    }, headers=headers)
    assert_true(status in (404, 405), f"expected 404 or 405, got {status}")


@test("Admin can toggle invite-free signup without blocking valid invites")
def test_admin_toggles_signup_policy():
    headers = admin_headers()
    original = read_settings(headers)["allow_signup_without_invite"]

    try:
        update_settings(headers, False)

        blocked_user = unique_user()
        status, body = request("POST", "/auth/register", blocked_user)
        assert_eq(status, 403)
        assert_eq(body["error"], "an invite is required")

        status, invite = request("POST", "/auth/invite/generate", {
            "ttl": "1h",
            "max_uses": 1,
        }, headers=headers)
        assert_eq(status, 201)

        invited_user = unique_user()
        status, _ = request("POST", "/auth/register", {
            **invited_user,
            "invite": invite["token"],
        })
        assert_eq(status, 201)

        update_settings(headers, True)

        open_user = unique_user()
        status, _ = request("POST", "/auth/register", open_user)
        assert_eq(status, 201)
    finally:
        update_settings(headers, original)
