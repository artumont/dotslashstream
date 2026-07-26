from tests.runner import test, section, assert_eq, assert_in, assert_true
from tests.client import request, unique_user
import uuid

section("Registration")


@test("Register new user returns tokens")
def test_register_success():
    user = unique_user()
    status, body = request("POST", "/auth/register", user)
    assert_eq(status, 201)
    assert_in("access_token", body)
    assert_in("refresh_token", body)


@test("Register with missing fields returns 400")
def test_register_missing_fields():
    status, _ = request("POST", "/auth/register", {"username": "x"})
    assert_eq(status, 400)


@test("Register with empty body returns 400")
def test_register_empty():
    status, _ = request("POST", "/auth/register", {})
    assert_eq(status, 400)


@test("Register duplicate username returns 409")
def test_register_duplicate():
    user = unique_user()
    request("POST", "/auth/register", user)
    dup = {**user, "email": f"other_{uuid.uuid4().hex}@e2e.test"}
    status, _ = request("POST", "/auth/register", dup)
    assert_eq(status, 409)


@test("Register with special characters in username returns 400")
def test_register_special_chars():
    for invalid in ["user@name", "user name", "user.name", "user-name", "user_name", "user!name"]:
        status, body = request("POST", "/auth/register", {
            "username": invalid,
            "email": f"{uuid.uuid4().hex[:8]}@e2e.test",
            "password": "password123",
        })
        assert_eq(status, 400, f"expected 400 for username '{invalid}'")


@test("Register with alphanumeric username succeeds")
def test_register_alphanumeric():
    user = unique_user()
    user["username"] = f"User123{uuid.uuid4().hex[:6]}"
    status, body = request("POST", "/auth/register", user)
    assert_eq(status, 201)
    assert_in("access_token", body)
