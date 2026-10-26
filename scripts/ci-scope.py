#!/usr/bin/env python3
"""Select expensive CI jobs by changed paths, while always reporting the workflow."""

import json
import os
from pathlib import Path
import subprocess

SCOPES = ("go", "web", "docs", "installers", "hmos")


def scopes_for(paths):
    scopes = set()
    for path in paths:
        if path.startswith(".github/") or path == "scripts/ci-scope.py":
            return set(SCOPES)
        if path.startswith(("cmd/", "internal/", "tests/integration/", "tests/smoke/")) or path.endswith(".go") or path in ("go.mod", "go.sum", ".golangci.yml", ".goreleaser.yml"):
            scopes.update(("go", "installers", "hmos"))
        if path.startswith(("web/", "deployment/", "internal/target/")) or path in ("scripts/gen-platform-surfaces.go", ".goreleaser.yml"):
            scopes.add("web")
        if path.startswith("docs/") or path == "scripts/build-docs.py":
            scopes.add("docs")
        if path.startswith(("tests/install-scripts/", "tests/release-scripts/")) or path in (
            "web/public/install.sh", "web/public/install.ps1", "scripts/extract-init-binaries.sh", "scripts/download-release-archives.sh",
        ):
            scopes.add("installers")
        if path.startswith("scripts/harmonyos/"):
            scopes.add("hmos")
    return scopes


def main():
    event = json.loads(Path(os.environ["GITHUB_EVENT_PATH"]).read_text(encoding="utf-8"))
    event_name = os.environ["GITHUB_EVENT_NAME"]
    if event_name == "pull_request":
        base = event["pull_request"]["base"]["sha"]
    elif event_name == "push":
        base = event.get("before", "")
    else:
        base = ""
    if not base or set(base) == {"0"}:
        scopes = set(SCOPES)
    else:
        # A missing base (for example after a force push) must broaden checks.
        result = subprocess.run(["git", "diff", "--name-only", "-z", base, "HEAD"], capture_output=True)
        scopes = scopes_for(result.stdout.decode("utf-8").rstrip("\0").split("\0")) if result.returncode == 0 else set(SCOPES)
    with open(os.environ["GITHUB_OUTPUT"], "a", encoding="utf-8") as output:
        for scope in SCOPES:
            line = f"{scope}={str(scope in scopes).lower()}"
            print(line)
            print(line, file=output)


if __name__ == "__main__":
    main()
