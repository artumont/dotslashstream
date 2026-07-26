from tests.runner import test, section, assert_eq, assert_in, assert_true
from tests.client import request, unique_user

section("Login")


@test("Login with valid credentials returns tokens")
def test_login_success():
    user = unique_user()
    status, _ = request("POST", "/auth/register", user)
    assert_eq(status, 201, "registration failed")
    status, body = request("POST", "/auth/login", {
        "username": user["username"],
        "password": user["password"],
    })
    assert_eq(status, 200)
    assert_in("access_token", body)
    assert_in("refresh_token", body)


@test("Login with wrong password returns 401")
def test_login_wrong_password():
    user = unique_user()
    status, _ = request("POST", "/auth/register", user)
    assert_eq(status, 201, "registration failed")
    status, _ = request("POST", "/auth/login", {
        "username": user["username"],
        "password": "wrongpassword",
    })
    assert_eq(status, 401)


@test("Login with nonexistent user returns 401")
def test_login_nonexistent():
    status, _ = request("POST", "/auth/login", {
        "username": "nonexistentuserxyz",
        "password": "password",
    })
    assert_eq(status, 401)


@test("Login with special characters in username returns 400")
def test_login_special_chars():
    status, _ = request("POST", "/auth/login", {
        "username": "invalid_user",
        "password": "password",
    })
    assert_eq(status, 400)


@test("Login with empty body returns 400")
def test_login_empty():
    status, _ = request("POST", "/auth/login", {})
    assert_eq(status, 400)
