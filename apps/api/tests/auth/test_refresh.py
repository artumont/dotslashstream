from tests.runner import test, section, assert_eq, assert_in
from tests.client import request, register_and_login

section("Token Refresh")


@test("Refresh with valid token returns new tokens")
def test_refresh_success():
    user, access, refresh = register_and_login()
    status, body = request("POST", "/auth/refresh", {
        "refresh_token": refresh,
    })
    assert_eq(status, 200)
    assert_in("access_token", body)
    assert_in("refresh_token", body)
    assert body["access_token"] != access, "access token should change"
    assert body["refresh_token"] != refresh, "refresh token should change"


@test("Refresh with invalid token returns 401")
def test_refresh_invalid():
    status, _ = request("POST", "/auth/refresh", {
        "refresh_token": "invalid.token.here",
    })
    assert_eq(status, 401)


@test("Refresh with empty body returns 400")
def test_refresh_empty():
    status, _ = request("POST", "/auth/refresh", {})
    assert_eq(status, 400)
