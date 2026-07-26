from tests.runner import test, section, assert_eq, assert_true
from tests.client import request, raw_request

section("Edge Cases")


@test("POST with invalid JSON returns 400")
def test_invalid_json():
    status, _ = raw_request("POST", "/auth/login", data=b"not json",
                            headers={"Content-Type": "application/json"})
    assert_true(status in (400, 415), f"expected 400 or 415, got {status}")


@test("POST with empty body returns 400")
def test_empty_body():
    status, _ = request("POST", "/auth/login")
    assert_eq(status, 400)


@test("GET on POST-only route returns 405 or 404")
def test_wrong_method():
    status, _ = request("GET", "/auth/login")
    assert_true(status in (405, 404), f"expected 405 or 404, got {status}")


@test("Very long input does not crash server")
def test_long_input():
    long = "a" * 1000
    status, _ = request("POST", "/auth/register", {
        "username": long,
        "email": f"{long}@test.com",
        "password": long,
    })
    assert_true(status != 500, f"server crashed on long input: {status}")


@test("SQL injection in username does not crash server")
def test_sql_injection():
    status, _ = request("POST", "/auth/register", {
        "username": "'; DROP TABLE users; --",
        "email": "sqli@test.com",
        "password": "password",
    })
    assert_true(status != 500, f"server may be vulnerable: {status}")
