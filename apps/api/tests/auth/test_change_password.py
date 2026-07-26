from tests.runner import test, section, assert_eq, assert_in
from tests.client import request, register_and_login, auth_headers
import uuid

section("Change Password")


@test("Change password without auth returns 401")
def test_change_pw_no_auth():
    status, _ = request("POST", "/auth/change-password", {
        "old_password": "old",
        "new_password": "new",
    })
    assert_eq(status, 401)


@test("Change password with wrong old password returns 401")
def test_change_pw_wrong_old():
    user, access, _ = register_and_login()
    status, _ = request("POST", "/auth/change-password", {
        "old_password": "wrongold",
        "new_password": "newpassword123",
    }, headers=auth_headers(access))
    assert_eq(status, 401)


@test("Change password with valid credentials succeeds")
def test_change_pw_success():
    user, access, _ = register_and_login()
    new_pass = f"new_{uuid.uuid4().hex[:8]}"
    status, body = request("POST", "/auth/change-password", {
        "old_password": user["password"],
        "new_password": new_pass,
    }, headers=auth_headers(access))
    assert_eq(status, 200)
    assert_in("message", body)

    # Login with new password should work
    status2, _ = request("POST", "/auth/login", {
        "username": user["username"],
        "password": new_pass,
    })
    assert_eq(status2, 200)


@test("Change password with invalid token returns 401")
def test_change_pw_bad_token():
    status, _ = request("POST", "/auth/change-password", {
        "old_password": "old",
        "new_password": "new",
    }, headers=auth_headers("invalid.token.here"))
    assert_eq(status, 401)
