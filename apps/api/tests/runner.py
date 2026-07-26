"""Minimal test runner with file/function selection.

Usage:
    python3 -m tests                                # all tests
    python3 -m tests auth/test_login.py             # one file
    python3 -m tests auth::test_login_success       # one function
"""

import importlib.util
import os
import sys
import time

# ── Test State ────────────────────────────────────────────────────────────────

_results = {"passed": 0, "failed": 0, "skipped": 0}
_current_section = ""


def section(name):
    """Print a section header."""
    global _current_section
    _current_section = name
    print(f"\n── {name} ──")


def test(name):
    """Decorator to register a test function."""
    def decorator(fn):
        fn._test_name = name
        fn._test_section = _current_section
        return fn
    return decorator


class SkipTest(Exception):
    pass


def skip(reason=""):
    raise SkipTest(reason)


def assert_eq(a, b, msg=""):
    assert a == b, f"{msg}: expected {b!r}, got {a!r}" if msg else f"expected {b!r}, got {a!r}"


def assert_true(val, msg=""):
    assert val, msg or f"expected truthy, got {val!r}"


def assert_in(key, d, msg=""):
    assert key in d, f"{msg}: {key!r} not in {d!r}" if msg else f"{key!r} not in {d!r}"


# ── Runner ────────────────────────────────────────────────────────────────────

def _run_one(fn):
    """Run a single test function. Returns True if passed."""
    name = getattr(fn, "_test_name", fn.__name__)
    try:
        fn()
        _results["passed"] += 1
        print(f"  ✓ {name}")
        return True
    except SkipTest as e:
        _results["skipped"] += 1
        reason = str(e) or "skipped"
        print(f"  ⊘ {name} ({reason})")
        return True
    except AssertionError as e:
        _results["failed"] += 1
        print(f"  ✗ {name}")
        print(f"    {e}")
        return False
    except Exception as e:
        _results["failed"] += 1
        print(f"  ✗ {name}")
        print(f"    {type(e).__name__}: {e}")
        return False


def _discover_tests(module):
    """Extract test functions from a module (functions with _test_name attr)."""
    tests = []
    for name in sorted(dir(module)):
        obj = getattr(module, name)
        if callable(obj) and hasattr(obj, "_test_name"):
            tests.append(obj)
    return tests


def _load_module(filepath):
    """Load a Python file as a module with full package path."""
    test_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))  # tests/
    rel = os.path.relpath(filepath, test_dir)                       # auth/test_foo.py
    modname = rel.replace(os.sep, ".").removesuffix(".py")        # auth.test_foo
    # Ensure parent package is importable
    parent = modname.rsplit(".", 1)[0] if "." in modname else None
    if parent and parent not in sys.modules:
        import types
        pkg_path = os.path.join(test_dir, parent.replace(".", os.sep))
        pkg = types.ModuleType(parent)
        pkg.__path__ = [pkg_path]
        pkg.__package__ = parent
        sys.modules[parent] = pkg
    spec = importlib.util.spec_from_file_location(modname, filepath,
        submodule_search_locations=[] if parent else None)
    if spec is None or spec.loader is None:
        raise ImportError(f"Cannot load {filepath}")
    mod = importlib.util.module_from_spec(spec)
    mod.__package__ = parent or "tests"
    sys.modules[modname] = mod
    spec.loader.exec_module(mod)
    return mod


def _discover_files(test_dir):
    """Walk test_dir and return all test_*.py files, sorted."""
    files = []
    for root, dirs, fnames in os.walk(test_dir):
        # Skip __pycache__ and root-level non-test files
        dirs[:] = [d for d in dirs if d != "__pycache__"]
        for f in sorted(fnames):
            if f.startswith("test_") and f.endswith(".py"):
                files.append(os.path.join(root, f))
    return sorted(files)


def _resolve_target(test_dir, target):
    """Resolve a target string to a filepath and optional function filter."""
    filter_func = None
    if "::" in target:
        target, filter_func = target.split("::", 1)

    # Try as relative path from test_dir
    path = os.path.join(test_dir, target)
    if os.path.exists(path):
        return path, filter_func

    # Try with .py extension
    if not target.endswith(".py"):
        path = os.path.join(test_dir, target + ".py")
        if os.path.exists(path):
            return path, filter_func

    # Try searching subdirectories
    for root, dirs, fnames in os.walk(test_dir):
        for f in fnames:
            if f == target or f == target + ".py":
                return os.path.join(root, f), filter_func

    return os.path.join(test_dir, target), filter_func


def run(targets=None):
    """Run tests. targets is a list of 'file.py', 'dir/file.py', or 'file.py::func_name'."""
    # Check API is reachable
    from .client import request
    try:
        request("GET", "/")
    except Exception as e:
        print(f"✗ Cannot reach API at {os.environ.get('API_URL', 'http://localhost:8080')}")
        print(f"  {e}")
        print(f"\n  Start the API first: make dev")
        sys.exit(1)

    test_dir = os.path.dirname(os.path.abspath(__file__))
    file_filters = []  # list of (filepath, filter_func_or_None)

    if targets:
        for t in targets:
            path, func = _resolve_target(test_dir, t)
            file_filters.append((path, func))
    else:
        for f in _discover_files(test_dir):
            file_filters.append((f, None))

    start = time.time()

    for filepath, filter_func in file_filters:
        if not os.path.exists(filepath):
            print(f"\n✗ File not found: {os.path.relpath(filepath, test_dir)}")
            continue

        mod = _load_module(filepath)
        tests = _discover_tests(mod)

        if filter_func:
            tests = [t for t in tests if t.__name__ == filter_func]

        if tests:
            modname = os.path.relpath(filepath, test_dir)
            print(f"\n{'─' * 50}")
            print(f" {modname}")
            print(f"{'─' * 50}")

            for t in tests:
                _run_one(t)

    elapsed = time.time() - start
    total = _results["passed"] + _results["failed"] + _results["skipped"]

    print(f"\n{'─' * 50}")
    print(f" Results: {_results['passed']} passed, {_results['failed']} failed, "
          f"{_results['skipped']} skipped ({total} total)")
    print(f" Time:    {elapsed:.2f}s")
    print(f"{'─' * 50}")

    return 0 if _results["failed"] == 0 else 1
