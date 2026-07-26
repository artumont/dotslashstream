from tests.runner import test, section, assert_true
from tests.client import request

section("Health")


@test("Server is reachable")
def test_server_reachable():
    status, _ = request("GET", "/")
    assert_true(status < 500, f"server returned {status}")


@test("Unknown route returns non-500")
def test_unknown_route():
    status, _ = request("GET", "/nonexistent")
    assert_true(status != 500, "server error on unknown route")
