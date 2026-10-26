import importlib.util
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("ci_scope", Path(__file__).with_name("ci-scope.py"))
scope = importlib.util.module_from_spec(spec)
spec.loader.exec_module(scope)


class ScopeTest(unittest.TestCase):
    def test_docs_do_not_start_native_runners(self):
        self.assertEqual(scope.scopes_for(["docs/user-guide/src/basic-usage.md"]), {"docs"})

    def test_go_includes_installers_and_harmony(self):
        for path in ["internal/config/config.go", "internal/config/testdata/settings.toml", "cmd/cjv/completion.txt"]:
            self.assertEqual(scope.scopes_for([path]), {"go", "installers", "hmos"})

    def test_platform_contract_rechecks_browser(self):
        for path in ["internal/target/platforms.go", "scripts/gen-platform-surfaces.go", ".goreleaser.yml"]:
            self.assertEqual(scope.scopes_for([path]), {"go", "installers", "hmos", "web"})

    def test_installer_and_release_changes(self):
        for path in ["web/public/install.sh", "scripts/download-release-archives.sh", "tests/release-scripts/release-scripts.sh"]:
            self.assertIn("installers", scope.scopes_for([path]))

    def test_ci_changes_keep_full_coverage(self):
        self.assertEqual(scope.scopes_for([".github/workflows/ci.yml"]), set(scope.SCOPES))
        self.assertEqual(scope.scopes_for(["scripts/ci-scope.py"]), set(scope.SCOPES))

    def run_event(self, name, event, git_result=None):
        with tempfile.TemporaryDirectory() as directory:
            event_path = Path(directory) / "event.json"
            output_path = Path(directory) / "output"
            event_path.write_text(json.dumps(event), encoding="utf-8")
            env = {"GITHUB_EVENT_NAME": name, "GITHUB_EVENT_PATH": str(event_path), "GITHUB_OUTPUT": str(output_path)}
            with patch.dict(os.environ, env), patch.object(scope.subprocess, "run", return_value=git_result) as git:
                scope.main()
            outputs = dict(line.split("=") for line in output_path.read_text(encoding="utf-8").splitlines())
            return {key for key, value in outputs.items() if value == "true"}, git

    def test_pr_uses_base_commit_and_nul_delimited_paths(self):
        scopes, git = self.run_event("pull_request", {"pull_request": {"base": {"sha": "base"}}}, subprocess.CompletedProcess([], 0, b"docs/space name.md\0"))
        self.assertEqual(scopes, {"docs"})
        self.assertEqual(git.call_args.args[0], ["git", "diff", "--name-only", "-z", "base", "HEAD"])

    def test_missing_push_base_runs_all_checks(self):
        scopes, _ = self.run_event("push", {"before": "old"}, subprocess.CompletedProcess([], 1, b""))
        self.assertEqual(scopes, set(scope.SCOPES))

    def test_manual_weekly_and_initial_push_run_all_without_diff(self):
        for name, event in [("workflow_dispatch", {}), ("schedule", {}), ("push", {"before": "0" * 40})]:
            scopes, git = self.run_event(name, event)
            self.assertEqual(scopes, set(scope.SCOPES))
            git.assert_not_called()


if __name__ == "__main__":
    unittest.main()
