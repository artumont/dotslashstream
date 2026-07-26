from tests.runner import test, section, assert_eq, assert_in, assert_true
from tests.client import request, register_and_login, auth_headers

section("Invite System")


@test("Generate invite without auth returns 401")
def test_invite_no_auth():
    status, _ = request("POST", "/auth/invite/generate", {
        "ttl": "168h",
        "max_uses": 5,
    })
    assert_eq(status, 401)


@test("Generate invite without admin returns 403")
def test_invite_not_admin():
    user, access, _ = register_and_login()
    status, _ = request("POST", "/auth/invite/generate", {
        "ttl": "168h",
        "max_uses": 5,
    }, headers=auth_headers(access))
    assert_eq(status, 403)


@test("Generate invite with invalid TTL returns 400 or 403")
def test_invite_bad_ttl():
    user, access, _ = register_and_login()
    status, _ = request("POST", "/auth/invite/generate", {
        "ttl": "invalid",
        "max_uses": 5,
    }, headers=auth_headers(access))
    assert_true(status in (400, 403), f"expected 400 or 403, got {status}")


@test("Generate invite with max_uses < 1 returns 400 or 403")
def test_invite_bad_uses():
    user, access, _ = register_and_login()
    status, _ = request("POST", "/auth/invite/generate", {
        "ttl": "168h",
        "max_uses": 0,
    }, headers=auth_headers(access))
    assert_true(status in (400, 403), f"expected 400 or 403, got {status}")


@test("Register with invalid invite returns 403")
def test_register_bad_invite():
    from tests.client import unique_user
    user = unique_user()
    status, _ = request("POST", "/auth/register", {
        **user,
        "invite": "some.invalid.token",
    })
    assert_eq(status, 403)
