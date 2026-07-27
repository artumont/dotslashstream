#!/usr/bin/env python3
"""E2E test runner for the dotslashstream API.

Zero dependencies — Python stdlib only.

Usage:
    python3 -m tests                              # all tests
    python3 -m tests sections                     # list available sections
    python3 -m tests -s Login                     # run a section
    python3 -m tests -s Login -s Registration     # run multiple sections
    python3 -m tests -t 2                         # 2s cooldown between sections
    python3 -m tests -s "Rate Limiting" -t 3      # section + cooldown
    python3 -m tests --no-blacklist               # ignore blacklist
    python3 -m tests auth/test_login.py           # one file
    python3 -m tests auth::test_login_success     # one function
    API_URL=http://localhost:9090 python3 -m tests
"""

import os
import sys

# Add project root to path so test files can import client/runner
PROJECT_ROOT = os.path.dirname(
    os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
)
sys.path.insert(0, PROJECT_ROOT)

from tests.runner import _cli  # noqa: E402


if __name__ == "__main__":
    sys.exit(_cli())
