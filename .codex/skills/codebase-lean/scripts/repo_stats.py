#!/usr/bin/env python3
"""Print code-size stats for the repo in cwd (or paths given as argv)."""
import os
import sys
from collections import defaultdict

EXTS = {".ts", ".tsx", ".vue", ".go", ".js", ".mjs", ".proto", ".css"}
SKIP_DIRS = {"node_modules", "dist", "build", ".git", ".pnpm-store", "__pycache__",
             "data", "archive", ".github", ".qoder", ".tools", ".workbuddy", "gen"}
SKIP_FILES = {"package-lock.json", "pnpm-lock.yaml"}
MAX_BYTES = 1_000_000


def count_lines(path):
    with open(path, "r", encoding="utf-8", errors="replace") as f:
        return sum(1 for _ in f)


def main():
    roots = sys.argv[1:] or ["."]
    files = []
    for root in roots:
        for dirpath, dirnames, filenames in os.walk(root):
            dirnames[:] = [d for d in dirnames if d not in SKIP_DIRS]
            for name in filenames:
                if name in SKIP_FILES:
                    continue
                if name.endswith(".pb.go"):
                    continue
                if os.path.splitext(name)[1].lower() not in EXTS:
                    continue
                path = os.path.join(dirpath, name)
                try:
                    if os.path.getsize(path) > MAX_BYTES:
                        continue
                    files.append((count_lines(path), path))
                except OSError:
                    pass
    if not files:
        print("no source files found")
        return
    total = sum(n for n, _ in files)
    print(f"total: {len(files)} files, {total} lines")

    print("\ntop 15 files by lines:")
    for n, p in sorted(files, reverse=True)[:15]:
        print(f"{n:6d}  {p}")

    by_area = defaultdict(lambda: [0, 0])
    for n, p in files:
        parts = p.replace("\\", "/").split("/")
        if parts[0] in ("", "."):
            parts = parts[1:]
        area = parts[1] if len(parts) > 1 and parts[0] == "packages" else parts[0]
        by_area[area][0] += 1
        by_area[area][1] += n
    print("\ntotals by area:")
    for area, (cnt, ln) in sorted(by_area.items()):
        print(f"{area:12s} {cnt:5d} files {ln:7d} lines")


if __name__ == "__main__":
    main()
