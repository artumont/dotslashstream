"""Minimal test runner with file/function selection, sections, and blacklisting.

Usage:
    python3 -m tests                              # all tests
    python3 -m tests --list                       # list available sections
    python3 -m tests -s Login                     # run a section
    python3 -m tests -s Login -s Registration     # run multiple sections
    python3 -m tests -t 2                         # 2s cooldown between sections
    python3 -m tests -s "Rate Limiting" -t 3      # section + cooldown
    python3 -m tests --no-blacklist               # ignore blacklist
    python3 -m tests auth/test_login.py           # one file
    python3 -m tests auth::test_login_success     # one function
    API_URL=http://localhost:9090 python3 -m tests
"""

import argparse
import importlib.util
import os
import sys
import time

# ── Test State ────────────────────────────────────────────────────────────────

_results = {"passed": 0, "failed": 0, "skipped": 0}
_current_section = ""


def section(name):
    """Register a section header. All subsequent @test decorators belong to it."""
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


# ── Blacklist ─────────────────────────────────────────────────────────────────

def _load_blacklist():
    """Load the blacklist file. Returns (sections, tests, files) sets."""
    sections = set()
    tests = set()
    files = set()

    path = os.environ.get("E2E_BLACKLIST", "")
    if not path:
        default = os.path.join(os.path.dirname(os.path.abspath(__file__)), "blacklist.txt")
        if os.path.exists(default):
            path = default

    if not path or not os.path.exists(path):
        return sections, tests, files

    with open(path) as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            if line.startswith("section:"):
                sections.add(line[len("section:"):])
            elif line.startswith("test:"):
                tests.add(line[len("test:"):])
            elif line.startswith("file:"):
                files.add(line[len("file:"):])

    return sections, tests, files


_blacklist_sections, _blacklist_tests, _blacklist_files = _load_blacklist()

# Global flag — set by CLI parsing, checked by _is_blacklisted.
_no_blacklist = False


def _is_blacklisted(fn, filepath, test_dir):
    """Check if a test should be skipped by the blacklist."""
    if _no_blacklist:
        return False

    rel = os.path.relpath(filepath, test_dir)
    if rel in _blacklist_files:
        return True

    name = getattr(fn, "_test_name", fn.__name__)
    if name in _blacklist_tests:
        return True

    section_name = getattr(fn, "_test_section", "")
    if section_name in _blacklist_sections:
        return True

    return False


# ── Module Discovery ──────────────────────────────────────────────────────────

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
    test_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    rel = os.path.relpath(filepath, test_dir)
    modname = rel.replace(os.sep, ".").removesuffix(".py")
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

    path = os.path.join(test_dir, target)
    if os.path.exists(path):
        return path, filter_func

    if not target.endswith(".py"):
        path = os.path.join(test_dir, target + ".py")
        if os.path.exists(path):
            return path, filter_func

    for root, dirs, fnames in os.walk(test_dir):
        for f in fnames:
            if f == target or f == target + ".py":
                return os.path.join(root, f), filter_func

    return os.path.join(test_dir, target), filter_func


# ── Sections Index ────────────────────────────────────────────────────────────

def _build_section_index(test_dir):
    """Load every test file and return {section_name: [(test_fn, filepath), ...]}."""
    index = {}
    for filepath in _discover_files(test_dir):
        mod = _load_module(filepath)
        for t in _discover_tests(mod):
            sec = getattr(t, "_test_section", "")
            index.setdefault(sec, []).append((t, filepath))
    return index


# ── Runner ────────────────────────────────────────────────────────────────────

def _run_one(fn):
    """Run a single test function."""
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


def _print_summary(elapsed):
    total = _results["passed"] + _results["failed"] + _results["skipped"]
    print(f"\n{'─' * 50}")
    print(f" Results: {_results['passed']} passed, {_results['failed']} failed, "
          f"{_results['skipped']} skipped ({total} total)")
    print(f" Time:    {elapsed:.2f}s")
    print(f"{'─' * 50}")


def _run_section(section_name, entries, test_dir, cooldown):
    """Run all tests in a section, printing its header."""
    print(f"\n── {section_name} ──")

    executed = 0
    for t, filepath in entries:
        if _is_blacklisted(t, filepath, test_dir):
            _results["skipped"] += 1
            name = getattr(t, "_test_name", t.__name__)
            print(f"  ⊘ {name} (blacklisted)")
            continue
        _run_one(t)
        executed += 1

    if cooldown > 0 and executed > 0:
        print(f"\n  ⏳ cooldown {cooldown}s ...")
        time.sleep(cooldown)


def _list_sections(test_dir):
    """Print all sections and their test count, then exit."""
    index = _build_section_index(test_dir)
    print(f"\n{'Section':<25} {'Tests':>5}")
    print(f"{'─' * 32}")
    for sec in sorted(index):
        print(f"{sec:<25} {len(index[sec]):>5}")
    print(f"{'─' * 32}")
    print(f"{'Total':<25} {sum(len(v) for v in index.values()):>5}")
    print()


def run(targets=None, sections=None, cooldown=0, no_blacklist=False):
    """Main entry point.

    Args:
        targets:   list of file/function strings (backward compat)
        sections:  list of section names to run (None = all)
        cooldown:  seconds to sleep between sections
        no_blacklist: ignore blacklist entirely
    """
    global _no_blacklist
    _no_blacklist = no_blacklist

    from .client import request
    try:
        request("GET", "/")
    except Exception as e:
        print(f"✗ Cannot reach API at {os.environ.get('API_URL', 'http://localhost:8080')}")
        print(f"  {e}")
        print(f"\n  Start the API first: make dev")
        sys.exit(1)

    test_dir = os.path.dirname(os.path.abspath(__file__))

    # ── Section mode ──────────────────────────────────────────────────────────

    if sections:
        index = _build_section_index(test_dir)
        missing = [s for s in sections if s not in index]
        if missing:
            print(f"✗ Unknown section(s): {', '.join(missing)}")
            print(f"  Run 'python3 -m tests --list' to list available sections")
            return 1

        start = time.time()
        for i, sec in enumerate(sections):
            is_last = i == len(sections) - 1
            _run_section(sec, index[sec], test_dir, cooldown if not is_last else 0)
        _print_summary(time.time() - start)
        return 0 if _results["failed"] == 0 else 1

    # ── File / function mode (backward compat) ────────────────────────────────

    file_filters = []
    if targets:
        for t in targets:
            path, func = _resolve_target(test_dir, t)
            file_filters.append((path, func))
    else:
        for f in _discover_files(test_dir):
            file_filters.append((f, None))

    # Group tests by section for cooldown support.
    start = time.time()
    current_section = None
    section_entries = []

    def _flush_section():
        nonlocal current_section, section_entries
        if current_section and section_entries:
            _run_section(current_section, section_entries, test_dir, cooldown)
        section_entries = []

    for filepath, filter_func in file_filters:
        if not os.path.exists(filepath):
            print(f"\n✗ File not found: {os.path.relpath(filepath, test_dir)}")
            continue

        mod = _load_module(filepath)
        tests = _discover_tests(mod)

        if filter_func:
            tests = [t for t in tests if t.__name__ == filter_func]

        for t in tests:
            sec = getattr(t, "_test_section", "")
            if sec != current_section:
                _flush_section()
                current_section = sec
            section_entries.append((t, filepath))

    _flush_section()
    _print_summary(time.time() - start)
    return 0 if _results["failed"] == 0 else 1


# ── CLI ───────────────────────────────────────────────────────────────────────

def _build_parser():
    p = argparse.ArgumentParser(
        prog="python3 -m tests",
        description="E2E test runner for the dotslashstream API",
    )
    p.add_argument("targets", nargs="*", metavar="TARGET",
        help="file.py, dir/file.py, or file.py::func_name")

    p.add_argument("-s", "--section", action="append", dest="sections",
        metavar="NAME",
        help="run only this section (repeatable: -s Login -s Registration)")
    p.add_argument("-t", "--timeout", type=float, default=0, metavar="SECS",
        help="cooldown in seconds between sections (default: 0)")
    p.add_argument("--no-blacklist", action="store_true",
        help="ignore the blacklist file entirely")
    p.add_argument("-l", "--list", action="store_true", dest="list_sections",
        help="list all available sections and exit")

    return p


def _cli():
    parser = _build_parser()
    args = parser.parse_args()

    test_dir = os.path.dirname(os.path.abspath(__file__))

    if args.list_sections:
        _list_sections(test_dir)
        return 0

    return run(
        targets=args.targets if args.targets else None,
        sections=args.sections if args.sections else None,
        cooldown=args.timeout,
        no_blacklist=args.no_blacklist,
    )
