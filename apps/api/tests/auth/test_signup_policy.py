"""Auth edge cases: signup policy interaction tests.

Tests that verify registration behavior when allow_signup_without_invite
is toggled, and how invite validation interacts with policy state.
"""

import os
import uuid

from tests.runner import test, section, assert_eq, assert_in, assert_true, skip
from tests.client import request, unique_user, auth_headers

section("Auth Edge Cases")

ADMIN_USERNAME = os.environ.get("E2E_ADMIN_USERNAME", "admin")
ADMIN_PASSWORD = os.environ.get("E2E_ADMIN_PASSWORD", "admin123")


def admin_login():
    status, body = request("POST", "/auth/login", {
        "username": ADMIN_USERNAME,
        "password": ADMIN_PASSWORD,
    })
    if status != 200:
        skip("requires seeded admin; run: make seed")
    return auth_headers(body["access_token"])


def enable_open_signup(headers):
    request("PATCH", "/settings", {
        "allow_signup_without_invite": True,
    }, headers=headers)


def disable_open_signup(headers):
    request("PATCH", "/settings", {
        "allow_signup_without_invite": False,
    }, headers=headers)


def generate_invite(headers, max_uses=1):
    status, body = request("POST", "/auth/invite/generate", {
        "ttl": "1h",
        "max_uses": max_uses,
    }, headers=headers)
    assert_eq(status, 201)
    return body["token"]


# ── Signup disabled, no invite ──────────────────────────────────────────────

@test("Register without invite when signup disabled returns 403")
def test_signup_disabled_no_invite():
    headers = admin_login()
    enable_open_signup(headers)  # ensure we can register admin changes

    disable_open_signup(headers)
    try:
        user = unique_user()
        status, body = request("POST", "/auth/register", user)
        assert_eq(status, 403)
        assert_eq(body["error"], "an invite is required")
    finally:
        enable_open_signup(headers)


@test("Register with empty invite string when signup disabled returns 403")
def test_signup_disabled_empty_invite():
    headers = admin_login()

    disable_open_signup(headers)
    try:
        user = unique_user()
        status, body = request("POST", "/auth/register", {
            **user,
            "invite": "",
        })
        assert_eq(status, 403)
        assert_eq(body["error"], "an invite is required")
    finally:
        enable_open_signup(headers)


@test("Register with invalid invite when signup disabled returns 403")
def test_signup_disabled_invalid_invite():
    headers = admin_login()

    disable_open_signup(headers)
    try:
        user = unique_user()
        status, body = request("POST", "/auth/register", {
            **user,
            "invite": "totally-fake-token-" + uuid.uuid4().hex,
        })
        assert_eq(status, 403)
    finally:
        enable_open_signup(headers)


# ── Signup disabled, valid invite ───────────────────────────────────────────

@test("Register with valid invite succeeds even when signup disabled")
def test_signup_disabled_valid_invite():
    headers = admin_login()

    disable_open_signup(headers)
    try:
        invite_token = generate_invite(headers)
        user = unique_user()
        status, body = request("POST", "/auth/register", {
            **user,
            "invite": invite_token,
        })
        assert_eq(status, 201)
        assert_in("access_token", body)
    finally:
        enable_open_signup(headers)


# ── Signup enabled, various invite states ───────────────────────────────────

@test("Register without invite when signup enabled succeeds")
def test_signup_enabled_no_invite():
    headers = admin_login()

    enable_open_signup(headers)
    user = unique_user()
    status, body = request("POST", "/auth/register", user)
    assert_eq(status, 201)
    assert_in("access_token", body)


@test("Register with invalid invite when signup enabled returns 403")
def test_signup_enabled_invalid_invite():
    headers = admin_login()

    enable_open_signup(headers)
    user = unique_user()
    status, body = request("POST", "/auth/register", {
        **user,
        "invite": "bad-token-" + uuid.uuid4().hex,
    })
    assert_eq(status, 403)


@test("Register with exhausted invite returns 403")
def test_signup_exhausted_invite():
    headers = admin_login()

    enable_open_signup(headers)
    invite_token = generate_invite(headers, max_uses=1)

    # First use succeeds
    user1 = unique_user()
    status, _ = request("POST", "/auth/register", {
        **user1,
        "invite": invite_token,
    })
    assert_eq(status, 201)

    # Second use fails
    user2 = unique_user()
    status, body = request("POST", "/auth/register", {
        **user2,
        "invite": invite_token,
    })
    assert_eq(status, 403)


# ── Policy toggle during session ────────────────────────────────────────────

@test("Toggling signup policy affects subsequent registrations")
def test_toggle_affects_new_registrations():
    headers = admin_login()

    # Start open
    enable_open_signup(headers)

    user1 = unique_user()
    status, _ = request("POST", "/auth/register", user1)
    assert_eq(status, 201)

    # Close signup
    disable_open_signup(headers)

    user2 = unique_user()
    status, body = request("POST", "/auth/register", user2)
    assert_eq(status, 403)
    assert_eq(body["error"], "an invite is required")

    # Reopen
    enable_open_signup(headers)

    user3 = unique_user()
    status, _ = request("POST", "/auth/register", user3)
    assert_eq(status, 201)


# ── Register response shape ─────────────────────────────────────────────────

@test("Register response contains user fields")
def test_register_response_shape():
    headers = admin_login()
    enable_open_signup(headers)

    user = unique_user()
    status, body = request("POST", "/auth/register", user)
    assert_eq(status, 201)
    assert_in("access_token", body)
    assert_in("refresh_token", body)
    assert_true(isinstance(body["access_token"], str))
    assert_true(isinstance(body["refresh_token"], str))
    assert_true(len(body["access_token"]) > 20)
    assert_true(len(body["refresh_token"]) > 20)



@test("Register with very long password returns 400")
def test_register_long_password():
    user = unique_user()
    status, body = request("POST", "/auth/register", {
        **user,
        "password": "a" * 200,
    })
    assert_eq(status, 400)


@test("Register with very long username returns 400")
def test_register_long_username():
    user = unique_user()
    status, body = request("POST", "/auth/register", {
        **user,
        "username": "a" * 200,
    })
    assert_eq(status, 400)


@test("Register with very long email returns 400")
def test_register_long_email():
    user = unique_user()
    status, body = request("POST", "/auth/register", {
        **user,
        "email": "a" * 300 + "@test.com",
    })
    assert_eq(status, 400)
