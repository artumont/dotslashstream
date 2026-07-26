#!/usr/bin/env python3
"""E2E test runner for the dotslashstream API.

Zero dependencies — Python stdlib only.

Usage:
    python3 tests/main.py                          # all tests
    python3 tests/main.py test_register.py         # one file
    python3 tests/main.py test_login.py::test_valid # one function
    API_URL=http://localhost:9090 python3 tests/main.py
"""

import os
import sys

# Add project root to path so test files can import client/runner
PROJECT_ROOT = os.path.dirname(
    os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
)
sys.path.insert(0, PROJECT_ROOT)

from tests.runner import run  # noqa: E402


if __name__ == "__main__":
    targets = sys.argv[1:] if len(sys.argv) > 1 else None
    sys.exit(run(targets))
