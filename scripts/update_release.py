#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# dependencies = ["tomlkit"]
# ///
"""Update release.env and cjpm.toml with the specified or latest release version."""

from __future__ import annotations

import argparse
import subprocess
import sys
from pathlib import Path

import tomlkit


def get_latest_tag(repo: str) -> str:
    result = subprocess.run(
        ["gh", "release", "view", "--repo", repo, "--json", "tagName", "-q", ".tagName"],
        capture_output=True,
        text=True,
        check=True,
    )
    tag = result.stdout.strip()
    if not tag:
        print("error: failed to fetch latest release tag", file=sys.stderr)
        sys.exit(1)
    return tag


def update_release_env(path: Path, version: str, tag: str, repo: str) -> None:
    base_url = f"https://github.com/{repo}/releases/download"
    updates = {
        "CJV_VERSION": version,
        "CJV_TAG": tag,
        "CJV_RELEASE_BASE_URL": base_url,
    }

    lines = path.read_text(encoding="utf-8").splitlines()
    out = []
    for line in lines:
        key, _, _ = line.partition("=")
        if key in updates:
            out.append(f"{key}={updates[key]}")
        else:
            out.append(line)
    path.write_text("\n".join(out) + "\n", encoding="utf-8")


def update_cjpm_toml(path: Path, version: str) -> None:
    doc = tomlkit.parse(path.read_text(encoding="utf-8"))
    doc["package"]["version"] = version  # type: ignore[index]
    path.write_text(tomlkit.dumps(doc), encoding="utf-8")


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Update release.env and cjpm.toml with version info.",
    )
    parser.add_argument(
        "--version",
        help="Version string (e.g. 0.1.8 or v0.1.8). "
        "If omitted, the latest release is fetched from GitHub via `gh`.",
    )
    parser.add_argument(
        "--dir",
        type=Path,
        default=Path("packaging/cjv"),
        help="Package directory containing release.env and cjpm.toml "
        "(default: packaging/cjv).",
    )
    parser.add_argument(
        "--repo",
        default="Zxilly/cjv",
        help="GitHub repository for auto-fetch (default: Zxilly/cjv).",
    )
    args = parser.parse_args()

    if args.version:
        version = args.version.lstrip("v")
        tag = f"v{version}"
    else:
        tag = get_latest_tag(args.repo)
        version = tag.lstrip("v")

    pkg_dir: Path = args.dir
    release_env = pkg_dir / "release.env"
    cjpm_toml = pkg_dir / "cjpm.toml"

    if not release_env.exists():
        print(f"error: {release_env} not found", file=sys.stderr)
        sys.exit(1)
    if not cjpm_toml.exists():
        print(f"error: {cjpm_toml} not found", file=sys.stderr)
        sys.exit(1)

    update_release_env(release_env, version, tag, args.repo)
    update_cjpm_toml(cjpm_toml, version)
    print(f"Updated {pkg_dir} to version {version} (tag: {tag})")


if __name__ == "__main__":
    main()
