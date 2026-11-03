#!/usr/bin/env python3
"""ArchLens parser for Python.

Reads a single .py file, finds its static imports, keeps only those that
resolve to something inside the project root, and prints JSON to stdout:

    {"package": str, "dependencies": [{"package": str, "file": str}, ...]}

Naming (package-level nodes):
  * A package is the directory containing a source file, written as its path
    relative to the root with "/" as the separator on every platform.
    The root directory itself is ".".
  * `package` is the package of the parsed file. Each dependency's `package`
    is the package of the file or directory the import resolved to.
  * A dependency on the parsed file's own package is omitted.
  * Names come from the resolved path, never from the import text, so the same
    file always gets the same string however it was imported.

The imported file (`file` of each dependency):
  * `file` is the file the import resolved to, as a path relative to the root
    with "/" as the separator: `import a.b` gives "a/b.py", and importing a
    package gives its "__init__.py".
  * `from pkg import name` gives "pkg/name.py" if `name` is a module, else
    the file of `pkg` itself (so a package that re-exports `name` gives its
    "__init__.py", not the file where `name` was defined).
  * A directory without an `__init__.py` (a namespace package) has no entry
    file, so `file` is the package path, the same string as `package`.
  * Dependencies are unique by (package, file): two different files of one
    package are two entries, the same file imported twice is one.

Resolution follows Python's own lookup rules:
  * absolute imports are searched in the directory of the parsed file (Python
    puts a script's own directory on sys.path), the project root, and
    <root>/src if it exists;
  * relative imports are resolved against the location of the parsed file;
  * regular modules/packages win over bare directories (namespace packages),
    and bare directories never shadow a standard library module;
  * `from pkg import name` checks whether `pkg.name` is a module first and
    falls back to `pkg`;
  * paths are resolved through symlinks before the "inside the root" check.

All static imports count, wherever they appear in the file (inside functions,
conditionals, TYPE_CHECKING blocks). Dynamic imports (importlib, __import__)
are ignored. Dependencies are listed in the order first found, without
duplicates. Errors go to stderr with a non-zero exit code.

Usage:
    python archlens_parser.py path/to/file.py [-r path/to/project]

The file path is resolved against the working directory (not the root).
Requires Python 3.10+ (sys.stdlib_module_names).
"""
from __future__ import annotations

import argparse
import ast
import json
import sys
from pathlib import Path

STDLIB = getattr(sys, "stdlib_module_names", frozenset())


def within(path: Path, root: Path) -> bool:
    return path == root or root in path.parents


def package_name(target: Path, root: Path) -> str | None:
    """Package containing `target` (a file's directory, or the directory itself).

    '/'-separated path relative to root, '.' for the root. None if outside it.
    """
    pkg_dir = target if target.is_dir() else target.parent
    if not within(pkg_dir, root):
        return None
    return pkg_dir.relative_to(root).as_posix()


def resolve_module(module: str, dirs: list[Path]) -> Path | None:
    """Resolve a dotted module name to a file or directory, or None."""
    parts = module.split(".") if module else []

    # 1. Regular modules and packages, in search order.
    for base in dirs:
        p = base.joinpath(*parts)
        candidates = [p / "__init__.py"]
        if parts:
            candidates.insert(0, p.parent / f"{p.name}.py")
        for c in candidates:
            if c.is_file():
                return c.resolve()

    # 2. Bare directories (namespace packages), never shadowing the stdlib.
    if parts and parts[0] not in STDLIB:
        for base in dirs:
            p = base.joinpath(*parts)
            if p.is_dir():
                return p.resolve()
    return None


def resolve_import(name: str, dirs: list[Path]) -> Path | None:
    """`import a.b.c` -> a.b.c if it resolves, else the longest resolving prefix."""
    parts = name.split(".")
    for i in range(len(parts), 0, -1):
        hit = resolve_module(".".join(parts[:i]), dirs)
        if hit is not None:
            return hit
    return None


class ImportCollector(ast.NodeVisitor):
    """Collects internal (package, file) pairs in source order, without duplicates."""

    def __init__(self, root: Path, file_dir: Path):
        self.root = root
        self.file_dir = file_dir
        dirs = [file_dir, root]
        if (root / "src").is_dir():
            dirs.append(root / "src")
        self.abs_dirs = list(dict.fromkeys(dirs))  # dedupe, keep order
        self.deps: dict[tuple[str, str], None] = {}  # insertion-ordered set

    def add(self, hit: Path | None) -> None:
        if hit is None:
            return
        if hit.is_dir():
            # A package is imported through its entry file, if it has one.
            entry = hit / "__init__.py"
            if entry.is_file():
                hit = entry.resolve()
        package = package_name(hit, self.root)
        if package is None:  # None means it resolved outside the root
            return
        if hit.is_dir():
            file = package  # namespace package: no entry file, so use the package path
        else:
            file = hit.relative_to(self.root).as_posix()
        self.deps[(package, file)] = None

    def visit_Import(self, node: ast.Import) -> None:
        for alias in node.names:
            self.add(resolve_import(alias.name, self.abs_dirs))

    def visit_ImportFrom(self, node: ast.ImportFrom) -> None:
        if node.level == 0:
            if not node.module:
                return
            base, dirs = node.module, self.abs_dirs
            base_hit = resolve_module(base, dirs)
        else:
            base_dir = self.file_dir
            for _ in range(node.level - 1):
                base_dir = base_dir.parent
            if not within(base_dir, self.root):
                return  # relative import climbs out of the project
            base, dirs = node.module or "", [base_dir]
            # `from . import x`: the base is the directory itself
            base_hit = resolve_module(base, dirs) if base else base_dir

        for alias in node.names:
            # `from pkg import name`: module first, fall back to the enclosing module
            sub = None
            if alias.name != "*":
                sub = resolve_module(f"{base}.{alias.name}" if base else alias.name, dirs)
            self.add(sub or base_hit)


def main() -> int:
    ap = argparse.ArgumentParser(description="ArchLens Python parser")
    ap.add_argument("file", type=Path)
    ap.add_argument("-r", "--root", type=Path, default=Path("."),
                    help="project root used to decide what is internal (default: .)")
    args = ap.parse_args()

    if not STDLIB:
        print("error: Python 3.10+ is required (sys.stdlib_module_names)", file=sys.stderr)
        return 1

    root = args.root.resolve()
    path = args.file.resolve()
    if not root.is_dir():
        print(f"error: root {root} is not a directory", file=sys.stderr)
        return 1
    if not within(path, root):
        print(f"error: {path} is not under root {root}", file=sys.stderr)
        return 1

    try:
        tree = ast.parse(path.read_bytes(), filename=str(path))
    except (SyntaxError, ValueError, OSError, RecursionError) as e:
        print(f"error: cannot parse {path}: {e}", file=sys.stderr)
        return 1

    package = package_name(path, root)

    collector = ImportCollector(root, path.parent)
    collector.visit(tree)
    deps = [{"package": p, "file": f} for (p, f) in collector.deps if p != package]

    json.dump({"package": package, "dependencies": deps}, sys.stdout)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    sys.exit(main())