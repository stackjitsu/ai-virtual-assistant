"""Real regression tests of the check adapter, independent of app runtime."""

import copy
import importlib.util
import json
from pathlib import Path
import tempfile
import sys
import unittest
from unittest.mock import patch


SPEC = importlib.util.spec_from_file_location("project_check", Path(__file__).with_name("check.py"))
check = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(check)


class CheckAdapterTests(unittest.TestCase):
    def temporary(self):
        return tempfile.TemporaryDirectory(dir="/tmp/opencode")

    def fixture(self, root):
        manifest = json.loads((check.ROOT / "harness.json").read_text())
        (root / "harness.json").write_text(json.dumps(manifest) + "\n")
        (root / "Makefile").write_text((check.ROOT / "Makefile").read_text())
        refs = [manifest["architecture"]["contextMap"]]
        refs.extend(c["contract"] for c in manifest["architecture"]["contexts"])
        refs.extend(["docs/contracts/chat.md", "docs/contracts/backend.md"])
        for ref in refs:
            path = root / ref
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text("# Contract\n")
        return manifest

    def test_manifest_accepts_baseline(self):
        with self.temporary() as directory:
            root = Path(directory)
            self.fixture(root)
            check.validate_manifest(root)

    def test_manifest_rejects_missing_command_and_overlap(self):
        with self.temporary() as directory:
            root = Path(directory)
            baseline = self.fixture(root)
            missing = copy.deepcopy(baseline)
            del missing["validation"]["commands"]["fuzz-full"]
            overlap = copy.deepcopy(baseline)
            overlap["architecture"]["contexts"][1]["ownedPaths"].append("web/assets/**")
            for changed in (missing, overlap):
                (root / "harness.json").write_text(json.dumps(changed) + "\n")
                with self.assertRaises(check.CheckError):
                    check.validate_manifest(root)

    def test_format_is_nonmutating_and_checks_newline(self):
        with self.temporary() as directory:
            root = Path(directory)
            path = root / "example.js"
            path.write_text("const n = 1; \n")
            before = path.read_bytes()
            with self.assertRaises(check.CheckError):
                check.format_check(root)
            self.assertEqual(path.read_bytes(), before)
            path.write_text("const n = 1;")
            with self.assertRaises(check.CheckError):
                check.format_check(root)
            path.write_text("const n = 1;\n")
            check.format_check(root)

    def test_gofmt_failure_does_not_write_source(self):
        with self.temporary() as directory:
            root = Path(directory)
            path = root / "example.go"
            path.write_text("package example\n")
            with patch.object(check, "run", return_value="example.go\n") as command:
                with self.assertRaises(check.CheckError):
                    check.format_check(root)
                self.assertEqual(command.call_args.args[0], ["gofmt", "-l", "example.go"])
            self.assertEqual(path.read_text(), "package example\n")

    def test_missing_packages_is_failure(self):
        with patch.object(check, "run", return_value=""):
            with self.assertRaisesRegex(check.CheckError, "packages missing"):
                check.go_packages()

    def test_command_failure_propagates_and_live_credentials_are_not_inherited(self):
        with patch.dict(check.os.environ, {"OPENAI_API_KEY": "fixture-only",
                                          "TURSO_AUTH_TOKEN": "fixture-only"}):
            check.run([sys.executable, "-c",
                       "import os; assert os.environ['OPENAI_API_KEY'] == ''; "
                       "assert os.environ['TURSO_AUTH_TOKEN'] == ''"], capture=True)
        with self.assertRaisesRegex(check.CheckError, "command failed \\(7\\)"):
            check.run([sys.executable, "-c", "raise SystemExit(7)"], capture=True)

    def test_fuzz_discovers_exact_targets_and_budgets(self):
        targets = [("ai-virtual-assistant/internal/chat", name) for name in check.MANDATORY_FUZZ]
        targets.append(("ai-virtual-assistant/internal/chat", "FuzzAnotherBoundary"))
        for seconds in (2, 30):
            plan = check.fuzz_plan(targets, seconds)
            self.assertEqual(len(plan), 4)
            self.assertTrue(all(f"-fuzztime={seconds}s" in command for command in plan))
            self.assertTrue(all(command[4].startswith("-fuzz=^Fuzz")
                                and command[4].endswith("$") for command in plan))
        with self.assertRaises(check.CheckError):
            check.fuzz_plan([], 2)
        with self.assertRaises(check.CheckError):
            check.fuzz_plan(targets[:1], 2)

    def test_go_target_listing_ignores_package_output(self):
        with patch.object(check, "go_packages", return_value=["example/chat"]):
            with patch.object(check, "run", return_value="FuzzInput\nok example/chat 0.1s\n"):
                self.assertEqual(check.go_targets("Fuzz"), [("example/chat", "FuzzInput")])

    def test_node_suites_require_real_owned_test_files(self):
        with self.temporary() as directory:
            root = Path(directory)
            (root / "web").mkdir()
            with self.assertRaises(check.CheckError):
                check.node_tests(root=root)
            (root / "web/widget.test.mjs").write_text("import 'node:test';\n")
            (root / "web/widget.contract.test.mjs").write_text("import 'node:test';\n")
            with patch.object(check, "run") as command:
                check.node_tests(root=root)
                self.assertEqual(command.call_args.args[0], ["node", "--test", "web/widget.test.mjs"])
                check.node_tests(contract=True, root=root)
                self.assertEqual(command.call_args.args[0], ["node", "--test", "web/widget.contract.test.mjs"])

    def test_source_discovery_excludes_shared_state(self):
        with self.temporary() as directory:
            root = Path(directory)
            for ref in (".harness/work.json", ".scratch/draft.md", "web/assets/widget.js"):
                path = root / ref
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text("\n")
            self.assertEqual([str(p.relative_to(root)) for p in check.files(root)], ["web/assets/widget.js"])


if __name__ == "__main__":
    unittest.main()
